package replay

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liraxapp/ortelius/utils"

	"github.com/liraxapp/avalanchego/ids"
	avlancheGoUtils "github.com/liraxapp/avalanchego/utils"
	"github.com/liraxapp/ortelius/cfg"
	"github.com/liraxapp/ortelius/services"
	"github.com/liraxapp/ortelius/services/indexes/avm"
	"github.com/liraxapp/ortelius/services/indexes/pvm"
	"github.com/liraxapp/ortelius/stream"
	"github.com/liraxapp/ortelius/stream/consumers"
	"github.com/segmentio/kafka-go"
)

type Replay interface {
	Start() error
}

func New(config *cfg.Config) Replay {
	return &replay{config: config,
		counterRead:  utils.NewCounterID(),
		counterAdded: utils.NewCounterID(),
		uniqueID:     make(map[string]utils.UniqueID),
	}
}

type replay struct {
	uniqueIDLock sync.RWMutex
	uniqueID     map[string]utils.UniqueID

	errs    *avlancheGoUtils.AtomicInterface
	running *avlancheGoUtils.AtomicBool
	config  *cfg.Config

	counterRead  *utils.CounterID
	counterAdded *utils.CounterID
}

func (replay *replay) Start() error {
	cfg.PerformUpdates = true

	replay.errs = &avlancheGoUtils.AtomicInterface{}
	replay.running = &avlancheGoUtils.AtomicBool{}
	replay.running.SetValue(true)

	for _, chainID := range replay.config.Chains {
		err := replay.handleReader(chainID)
		if err != nil {
			log.Fatalln("reader failed", chainID, ":", err.Error())
			return err
		}
	}

	for replay.running.GetValue() {
		type CounterValues struct {
			Read  uint64
			Added uint64
		}

		ctot := make(map[string]*CounterValues)
		countersValues := replay.counterRead.Clone()
		for cnter := range countersValues {
			if _, ok := ctot[cnter]; !ok {
				ctot[cnter] = &CounterValues{}
			}
			ctot[cnter].Read = countersValues[cnter]
		}
		countersValues = replay.counterAdded.Clone()
		for cnter := range countersValues {
			if _, ok := ctot[cnter]; !ok {
				ctot[cnter] = &CounterValues{}
			}
			ctot[cnter].Added = countersValues[cnter]
		}

		for cnter := range ctot {
			replay.config.Services.Log.Info("key:%s read:%d add:%d", cnter, ctot[cnter].Read, ctot[cnter].Added)
		}

		time.Sleep(5 * time.Second)
	}

	if replay.errs.GetValue() != nil {
		replay.config.Services.Log.Info("replay failed %w", replay.errs.GetValue().(error))
		return replay.errs.GetValue().(error)
	}

	return nil
}

func (replay *replay) handleReader(chain cfg.Chain) error {
	conns, err := services.NewConnectionsFromConfig(replay.config.Services, false)
	if err != nil {
		return err
	}

	var writer services.Consumer
	switch chain.VMType {
	case consumers.IndexerAVMName:
		writer, err = avm.NewWriter(conns, replay.config.NetworkID, chain.ID)
		if err != nil {
			return err
		}
	case consumers.IndexerPVMName:
		writer, err = pvm.NewWriter(conns, replay.config.NetworkID, chain.ID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown vmtype")
	}

	uidkeyconsensus := fmt.Sprintf("%s:%s", stream.EventTypeConsensus, chain.ID)
	uidkeydecision := fmt.Sprintf("%s:%s", stream.EventTypeDecisions, chain.ID)

	replay.uniqueIDLock.Lock()
	replay.uniqueID[uidkeyconsensus] = utils.NewMemoryUniqueID()
	replay.uniqueID[uidkeydecision] = utils.NewMemoryUniqueID()
	replay.uniqueIDLock.Unlock()

	go func() {
		defer replay.running.SetValue(false)
		tn := stream.GetTopicName(replay.config.NetworkID, chain.ID, stream.EventTypeDecisions)

		reader := kafka.NewReader(kafka.ReaderConfig{
			Topic:       tn,
			Brokers:     replay.config.Kafka.Brokers,
			GroupID:     replay.config.Consumer.GroupName,
			StartOffset: kafka.FirstOffset,
			MaxBytes:    stream.ConsumerMaxBytesDefault,
		})

		ctx := context.Background()

		err := writer.Bootstrap(ctx)
		if err != nil {
			replay.errs.SetValue(err)
			return
		}

		for replay.running.GetValue() {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.counterRead.Inc(tn)

			id, err := ids.ToID(msg.Key)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.uniqueIDLock.RLock()
			present, err := replay.uniqueID[uidkeydecision].Get(id.String())
			replay.uniqueIDLock.RUnlock()
			if err != nil {
				replay.errs.SetValue(err)
				return
			}
			if present {
				continue
			}

			replay.counterAdded.Inc(tn)

			msgc := stream.NewMessage(
				id.String(),
				chain.ID,
				msg.Value,
				msg.Time.UTC().Unix(),
			)

			err = writer.Consume(msgc)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.uniqueIDLock.RLock()
			err = replay.uniqueID[uidkeydecision].Put(id.String())
			replay.uniqueIDLock.RUnlock()
			if err != nil {
				replay.errs.SetValue(err)
				return
			}
		}
	}()

	go func() {
		defer replay.running.SetValue(false)
		tn := stream.GetTopicName(replay.config.NetworkID, chain.ID, stream.EventTypeConsensus)

		reader := kafka.NewReader(kafka.ReaderConfig{
			Topic:       tn,
			Brokers:     replay.config.Kafka.Brokers,
			GroupID:     replay.config.Consumer.GroupName,
			StartOffset: kafka.FirstOffset,
			MaxBytes:    stream.ConsumerMaxBytesDefault,
		})

		ctx := context.Background()

		for replay.running.GetValue() {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.counterRead.Inc(tn)

			id, err := ids.ToID(msg.Key)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.uniqueIDLock.RLock()
			present, err := replay.uniqueID[uidkeyconsensus].Get(id.String())
			replay.uniqueIDLock.RUnlock()
			if err != nil {
				replay.errs.SetValue(err)
				return
			}
			if present {
				continue
			}

			replay.counterAdded.Inc(tn)

			msgc := stream.NewMessage(
				id.String(),
				chain.ID,
				msg.Value,
				msg.Time.UTC().Unix(),
			)

			err = writer.ConsumeConsensus(msgc)
			if err != nil {
				replay.errs.SetValue(err)
				return
			}

			replay.uniqueIDLock.RLock()
			err = replay.uniqueID[uidkeyconsensus].Put(id.String())
			replay.uniqueIDLock.RUnlock()
			if err != nil {
				replay.errs.SetValue(err)
				return
			}
		}
	}()

	return nil
}
