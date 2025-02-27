package proxynode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/milvus-io/milvus/internal/log"
	"go.uber.org/zap"
)

type pChanStatistics struct {
	minTs Timestamp
	maxTs Timestamp
}

// ticker can update ts only when the minTs greater than the ts of ticker, we can use maxTs to update current later
type getPChanStatisticsFuncType func() (map[pChan]*pChanStatistics, error)

// use interface tsoAllocator to keep channelsTimeTickerImpl testable
type tsoAllocator interface {
	//Start() error
	AllocOne() (Timestamp, error)
	//Alloc(count uint32) ([]Timestamp, error)
	//ClearCache()
}

type channelsTimeTicker interface {
	start() error
	close() error
	addPChan(pchan pChan) error
	removePChan(pchan pChan) error
	getLastTick(pchan pChan) (Timestamp, error)
	getMinTsStatistics() (map[pChan]Timestamp, error)
}

type channelsTimeTickerImpl struct {
	interval          time.Duration       // interval to synchronize
	minTsStatistics   map[pChan]Timestamp // pchan -> min Timestamp
	statisticsMtx     sync.RWMutex
	getStatisticsFunc getPChanStatisticsFuncType
	tso               tsoAllocator
	currents          map[pChan]Timestamp
	currentsMtx       sync.RWMutex
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
}

func (ticker *channelsTimeTickerImpl) getMinTsStatistics() (map[pChan]Timestamp, error) {
	ticker.statisticsMtx.RLock()
	defer ticker.statisticsMtx.RUnlock()

	ret := make(map[pChan]Timestamp)
	for k, v := range ticker.minTsStatistics {
		if v > 0 {
			ret[k] = v
		}
	}
	return ret, nil
}

func (ticker *channelsTimeTickerImpl) initStatistics() {
	ticker.statisticsMtx.Lock()
	defer ticker.statisticsMtx.Unlock()

	for pchan := range ticker.minTsStatistics {
		ticker.minTsStatistics[pchan] = 0
	}
}

func (ticker *channelsTimeTickerImpl) initCurrents(current Timestamp) {
	ticker.currentsMtx.Lock()
	defer ticker.currentsMtx.Unlock()

	for pchan := range ticker.currents {
		ticker.currents[pchan] = current
	}
}

func (ticker *channelsTimeTickerImpl) tick() error {
	now, err := ticker.tso.AllocOne()
	if err != nil {
		log.Warn("ProxyNode channelsTimeTickerImpl failed to get ts from tso", zap.Error(err))
		return err
	}
	//nowPTime, _ := tsoutil.ParseTS(now)

	ticker.statisticsMtx.Lock()
	defer ticker.statisticsMtx.Unlock()

	ticker.currentsMtx.Lock()
	defer ticker.currentsMtx.Unlock()

	stats, err := ticker.getStatisticsFunc()
	if err != nil {
		log.Debug("ProxyNode channelsTimeTickerImpl failed to getStatistics", zap.Error(err))
	}

	for pchan := range ticker.currents {
		current := ticker.currents[pchan]
		//currentPTime, _ := tsoutil.ParseTS(current)
		stat, ok := stats[pchan]
		//log.Debug("ProxyNode channelsTimeTickerImpl", zap.Any("pchan", pchan),
		//	zap.Any("TaskInQueue", ok),
		//	zap.Any("current", currentPTime),
		//	zap.Any("now", nowPTime))
		if !ok {
			ticker.minTsStatistics[pchan] = current
			ticker.currents[pchan] = now
		} else {
			//minPTime, _ := tsoutil.ParseTS(stat.minTs)
			//maxPTime, _ := tsoutil.ParseTS(stat.maxTs)

			if stat.minTs > current {
				cur := current
				if stat.minTs > now {
					cur = now
				}
				ticker.minTsStatistics[pchan] = cur
				next := now + Timestamp(sendTimeTickMsgInterval)
				if next > stat.maxTs {
					next = stat.maxTs
				}
				ticker.currents[pchan] = next
				//nextPTime, _ := tsoutil.ParseTS(next)
				//log.Debug("ProxyNode channelsTimeTickerImpl",
				//	zap.Any("pchan", pchan),
				//	zap.Any("minPTime", minPTime),
				//	zap.Any("maxPTime", maxPTime),
				//	zap.Any("nextPTime", nextPTime))
			}
		}
	}

	return nil
}

func (ticker *channelsTimeTickerImpl) tickLoop() {
	defer ticker.wg.Done()

	timer := time.NewTicker(ticker.interval)
	defer timer.Stop()

	for {
		select {
		case <-ticker.ctx.Done():
			return
		case <-timer.C:
			err := ticker.tick()
			if err != nil {
				log.Warn("channelsTimeTickerImpl.tickLoop", zap.Error(err))
			}
		}
	}
}

func (ticker *channelsTimeTickerImpl) start() error {
	ticker.initStatistics()

	current, err := ticker.tso.AllocOne()
	if err != nil {
		return err
	}
	ticker.initCurrents(current)

	ticker.wg.Add(1)
	go ticker.tickLoop()

	return nil
}

func (ticker *channelsTimeTickerImpl) close() error {
	ticker.cancel()
	ticker.wg.Wait()
	return nil
}

func (ticker *channelsTimeTickerImpl) addPChan(pchan pChan) error {
	ticker.statisticsMtx.Lock()

	if _, ok := ticker.minTsStatistics[pchan]; ok {
		ticker.statisticsMtx.Unlock()
		return fmt.Errorf("pChan %v already exist in minTsStatistics", pchan)
	}
	ticker.minTsStatistics[pchan] = 0
	ticker.statisticsMtx.Unlock()

	ticker.currentsMtx.Lock()
	defer ticker.currentsMtx.Unlock()
	if _, ok := ticker.currents[pchan]; ok {
		return fmt.Errorf("pChan %v already exist in currents", pchan)
	}
	ticker.currents[pchan] = 0

	return nil
}

func (ticker *channelsTimeTickerImpl) removePChan(pchan pChan) error {
	ticker.statisticsMtx.Lock()

	if _, ok := ticker.minTsStatistics[pchan]; !ok {
		ticker.statisticsMtx.Unlock()
		return fmt.Errorf("pChan %v don't exist in minTsStatistics", pchan)
	}
	delete(ticker.minTsStatistics, pchan)
	ticker.statisticsMtx.Unlock()

	ticker.currentsMtx.Lock()
	defer ticker.currentsMtx.Unlock()

	if _, ok := ticker.currents[pchan]; !ok {
		return fmt.Errorf("pChan %v don't exist in currents", pchan)
	}
	delete(ticker.currents, pchan)

	return nil
}

func (ticker *channelsTimeTickerImpl) getLastTick(pchan pChan) (Timestamp, error) {
	ticker.statisticsMtx.RLock()
	defer ticker.statisticsMtx.RUnlock()

	ts, ok := ticker.minTsStatistics[pchan]
	if !ok {
		return 0, fmt.Errorf("pChan %v not found", pchan)
	}

	return ts, nil
}

func newChannelsTimeTicker(
	ctx context.Context,
	interval time.Duration,
	pchans []pChan,
	getStatisticsFunc getPChanStatisticsFuncType,
	tso tsoAllocator,
) *channelsTimeTickerImpl {

	ctx1, cancel := context.WithCancel(ctx)

	ticker := &channelsTimeTickerImpl{
		interval:          interval,
		minTsStatistics:   make(map[pChan]Timestamp),
		getStatisticsFunc: getStatisticsFunc,
		tso:               tso,
		currents:          make(map[pChan]Timestamp),
		ctx:               ctx1,
		cancel:            cancel,
	}

	for _, pchan := range pchans {
		ticker.minTsStatistics[pchan] = 0
		ticker.currents[pchan] = 0
	}

	return ticker
}
