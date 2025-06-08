package repository

import (
	"context"
	"github.com/db-mipt/game-client/internal/config"
	"github.com/db-mipt/game-client/internal/model"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PlayerRepository struct {
	log    *logrus.Entry
	cfg    config.REDIS
	client *redis.ClusterClient
	shards []string
}

func NewPlayerRepository(lc fx.Lifecycle,
	cfg *config.Config,
	log *logrus.Entry,
) *PlayerRepository {
	nodes := strings.Split(cfg.REDIS.ClusterAddress, ",")
	rep := &PlayerRepository{
		log: log,
		cfg: cfg.REDIS,
		client: redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:          nodes,
			MaxRedirects:   8,
			ReadOnly:       false, // Чтение только с мастеров (если true — может читать с реплик)
			RouteByLatency: false, // Не выбирать ноду по задержке (если true — может влиять на consistency)
			DialTimeout:    5 * time.Second,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   3 * time.Second,
			PoolSize:       10, // Пул соединений на каждую ноду
			// Критично для `cluster-require-full-coverage no`:
			MaxRetries:      2,
			MinRetryBackoff: 1 * time.Second,
			MaxRetryBackoff: 1 * time.Second,
		}),
	}

	err := rep.client.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	})

	if err != nil {
		panic(err)
	}

	for i, _ := range nodes {
		rep.shards = append(rep.shards, cfg.REDIS.LeaderBoardKeyBase+"_"+strconv.Itoa(i))
	}

	return rep
}

func (repository *PlayerRepository) UpdateScore(id string, score float64) error {
	ctx := context.Background()

	shardName, err := repository.getLeaderBoardShardName(id)
	if err != nil {
		return err
	}

	_, err = repository.client.ZAdd(ctx, shardName, redis.Z{
		Score:  score,
		Member: id,
	}).Result()

	return err
}

func (repository *PlayerRepository) GetTopNPlayers(count int64) ([]*model.RankedPlayer, error) {
	leaderBoard, err := repository.getLeaderBoard()
	if err != nil {
		return nil, err
	}

	return leaderBoard[:min(int64(len(leaderBoard)), count)], nil
}

func (repository *PlayerRepository) GetPlayerRank(id string) (*model.RankedPlayer, error) {
	leaderBoard, err := repository.getLeaderBoard()
	if err != nil {
		return nil, err
	}

	for _, player := range leaderBoard {
		if player.Id == id {
			return player, nil
		}
	}

	return nil, nil
}

func (repository *PlayerRepository) getLeaderBoard() ([]*model.RankedPlayer, error) {
	ctx := context.Background()

	players := make([]*model.RankedPlayer, 0)
	for _, leaderBoardShard := range repository.shards {
		if !repository.isLeaderBoardShardServedByCluster(leaderBoardShard) {
			repository.log.Infof("LeaderBoard shard %v not served by cluster", leaderBoardShard)
			continue
		}

		result, err := repository.client.ZRangeWithScores(ctx, leaderBoardShard, 0, -1).Result()
		if err != nil {
			return nil, err
		}

		for i, z := range result {
			playerId := z.Member.(string)

			players = append(players, &model.RankedPlayer{
				Id:    playerId,
				Score: z.Score,
				Rank:  int64(i + 1),
			})
		}
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].Score > players[j].Score
	})

	for i, player := range players {
		player.Rank = int64(i + 1)
	}

	repository.log.Infof("LeaderBoard state is: %v", players)

	return players, nil
}

func (repository *PlayerRepository) isLeaderBoardShardServedByCluster(leaderBoardShardName string) bool {
	ctx := context.Background()

	slot, err := repository.client.ClusterKeySlot(ctx, leaderBoardShardName).Result()
	if err != nil {
		return false
	}

	slots, err := repository.client.ClusterSlots(ctx).Result()
	if err != nil {
		return false
	}

	repository.log.Infof("Slots info: %v", slots)

	for _, s := range slots {
		if slot >= int64(s.Start) && slot <= int64(s.End) {
			if len(s.Nodes) > 0 {
				return true
			}
		}
	}

	return false
}

func (repository *PlayerRepository) getLeaderBoardShardName(key string) (string, error) {
	ctx := context.Background()

	slot, err := repository.client.ClusterKeySlot(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return repository.shards[slot%int64(len(repository.shards))], nil
}
