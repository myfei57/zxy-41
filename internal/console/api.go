package console

import (
	"context"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"waternet/internal/audit"
	"waternet/internal/dispatch"
	"waternet/internal/hammer"
	"waternet/internal/ns"
	"waternet/internal/policy"
	"waternet/internal/pressure"
	"waternet/internal/pump"
	"waternet/internal/quota"
	"waternet/internal/station"
	"waternet/internal/store"
	"waternet/internal/valve"
)

// Server wires every component of the water network behind one HTTP console.
type Server struct {
	fs         *store.FileStore
	registry   *ns.Registry
	station    *station.Station
	mapper     *pressure.ZoneMap
	sampler    *pressure.Sampler
	windows    *pressure.WindowStore
	agg        *store.WindowAggregator
	ledger     *quota.Ledger
	recorder   *audit.Recorder
	exec       *dispatch.Executor
	receipts   *store.ExecutionLog
	guard      *hammer.Guard
	valveExec  *valve.Executor
	cumulative *store.CumulativeLedger
	policyCfg  *policy.Config
	evaluator  *policy.Evaluator
	decider    *policy.Decider
	snapshots  *store.SnapshotService
	queue      *dispatch.OrderQueue

	mu        sync.Mutex
	seq       int64
	sessions  map[string]*dispatch.Session
	group1    *pump.Group
	group2    *pump.Group
	valveUp   *valve.Valve
	valveDown *valve.Valve
	valveIso  *valve.Valve
	meter1    pressure.Meter
	tank1     *station.Tank
}

// New assembles a server from already built components.
func New(
	fs *store.FileStore,
	registry *ns.Registry,
	st *station.Station,
	mapper *pressure.ZoneMap,
	sampler *pressure.Sampler,
	windows *pressure.WindowStore,
	agg *store.WindowAggregator,
	ledger *quota.Ledger,
	recorder *audit.Recorder,
	exec *dispatch.Executor,
	guard *hammer.Guard,
	valveExec *valve.Executor,
	cumulative *store.CumulativeLedger,
	policyCfg *policy.Config,
	evaluator *policy.Evaluator,
	decider *policy.Decider,
	snapshots *store.SnapshotService,
	queue *dispatch.OrderQueue,
) *Server {
	return &Server{
		fs:         fs,
		registry:   registry,
		station:    st,
		mapper:     mapper,
		sampler:    sampler,
		windows:    windows,
		agg:        agg,
		ledger:     ledger,
		recorder:   recorder,
		exec:       exec,
		guard:      guard,
		valveExec:  valveExec,
		cumulative: cumulative,
		policyCfg:  policyCfg,
		evaluator:  evaluator,
		decider:    decider,
		snapshots:  snapshots,
		queue:      queue,
		sessions:   map[string]*dispatch.Session{},
	}
}

// Router builds the chi route tree.
func (s *Server) Router() chi.Router {
	router := chi.NewRouter()
	s.routes(router)
	return router
}

// Session returns a named dispatch session, creating it on first use.
func (s *Server) Session(name string) *dispatch.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[name]; ok {
		return session
	}
	session := dispatch.NewSession(s.exec)
	s.sessions[name] = session
	return session
}

// NextSeq returns the next dispatch issue sequence number.
func (s *Server) NextSeq() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

// StateRecord renders the current network state for persistence.
func (s *Server) StateRecord() store.StateRecord {
	record := store.StateRecord{
		Version:    1,
		SnapshotAt: time.Now().UTC().Format(time.RFC3339Nano),
		ZoneMap:    s.mapper.Snapshot(),
	}
	for _, n := range s.registry.All() {
		record.Namespaces = append(record.Namespaces, store.NamespaceRecord{
			ID:    n.ID,
			Name:  n.Name,
			Zones: n.ZoneIDs(),
		})
	}
	record.Stations = append(record.Stations, s.station.Record())
	for _, group := range []*pump.Group{s.group1, s.group2} {
		if group != nil {
			record.Pumps = append(record.Pumps, group.Record())
		}
	}
	for _, v := range []*valve.Valve{s.valveUp, s.valveDown, s.valveIso} {
		if v != nil {
			record.Valves = append(record.Valves, v.Record())
		}
	}
	record.Quota = store.QuotaRecord{
		Zone:   s.station.ZoneID,
		Used:   s.ledger.Used(s.station.ZoneID),
		Limit:  s.ledger.Limit(),
		Window: s.ledger.Status(s.station.ZoneID).Window,
	}
	if target, err := s.policyCfg.TargetFor(s.station.ZoneID); err == nil {
		record.Policy = store.PolicyRecord{
			Zone:      target.Zone,
			TargetMPA: target.TargetMPA,
			MinMPA:    target.MinMPA,
			MaxMPA:    target.MaxMPA,
			Strategy:  target.Strategy,
		}
	}
	return record
}

// Tick runs one control cycle: refresh, sample, failover, evaluate and
// snapshot. It returns the first error encountered.
func (s *Server) Tick(ctx context.Context) error {
	if err := s.station.Refresh(ctx); err != nil {
		return err
	}
	if s.ledger.Status(s.station.ZoneID).Window != time.Now().UTC().Format("20060102") {
		s.ledger.Rollover()
	}
	for _, v := range []*valve.Valve{s.valveUp, s.valveDown, s.valveIso} {
		if v == nil {
			continue
		}
		if _, err := s.sampler.Sample(ctx, v.ID, 0.40, 10.0, time.Now().UTC()); err != nil {
			return err
		}
	}
	if _, err := pump.Failover(ctx, s.group1, s.station); err != nil {
		return err
	}
	_ = pump.SaveSnapshot(s.group1)
	result, err := s.evaluator.EvaluateZone(ctx, s.station.ZoneID)
	if err != nil {
		return err
	}
	command, err := s.decider.Decide(s.station.ZoneID, result, s.group1.ID, "u-1")
	if err == nil {
		if err := s.Session("scheduler").Send(ctx, command, s.group1); err != nil {
			return err
		}
	}
	_, err = s.snapshots.Write(s.StateRecord())
	return err
}

// Bootstrap creates the default water network and restores persisted state.
func Bootstrap(dataDir string) (*Server, error) {
	fs, err := store.New(dataDir)
	if err != nil {
		return nil, err
	}
	registry := ns.NewRegistry(fs)
	if registry.Load() != nil {
		downtown := ns.NewNamespace("downtown", "城东供水片区")
		if err := downtown.AddZone("north", "北区"); err != nil {
			return nil, err
		}
		if err := downtown.AddZone("south", "南区"); err != nil {
			return nil, err
		}
		if err := registry.Register(downtown); err != nil {
			return nil, err
		}
	}
	recorder := audit.NewRecorder(fs)
	st := station.New("st-1", "主泵站", "downtown", "north", fs, recorder)
	tank := station.NewTank("t-1", "清水池", 2000)
	if err := tank.SetLevel(800); err != nil {
		return nil, err
	}
	if err := st.AddTank(tank); err != nil {
		return nil, err
	}
	group1 := pump.NewGroup("g-1", "st-1", "主泵组", fs)
	group1.AddUnit("u-1", "1号主泵")
	group1.AddUnit("u-2", "1号备泵")
	group2 := pump.NewGroup("g-2", "st-1", "辅泵组", fs)
	group2.AddUnit("u-3", "2号主泵")
	group2.AddUnit("u-4", "2号备泵")
	if err := st.AttachGroup(group1); err != nil {
		return nil, err
	}
	if err := st.AttachGroup(group2); err != nil {
		return nil, err
	}
	valveUp := valve.NewValve("v-up", "st-1", "上游隔离阀", fs)
	valveDown := valve.NewValve("v-down", "st-1", "下游控制阀", fs)
	valveIso := valve.NewValve("v-iso", "st-1", "南北隔离阀", fs)
	valveIso.SetZone("south")
	for _, v := range []*valve.Valve{valveUp, valveDown, valveIso} {
		if err := st.AttachValve(v); err != nil {
			return nil, err
		}
	}
	mapper := pressure.NewZoneMap()
	if err := mapper.AttachZone(valveUp.ID, "north"); err != nil {
		return nil, err
	}
	if err := mapper.AttachZone(valveDown.ID, "north"); err != nil {
		return nil, err
	}
	if err := mapper.AttachZone(valveIso.ID, "south"); err != nil {
		return nil, err
	}
	agg := store.NewWindowAggregator(fs)
	windows := pressure.NewWindowStore(agg)
	ledger := quota.NewLedger(fs, 1000000)
	if ledger.Load() != nil {
		_ = ledger.Save()
	}
	sampler := pressure.NewSampler(mapper, agg, ledger, recorder, fs)
	cumulative := store.NewCumulativeLedger(fs)
	if cumulative.Load() != nil {
		_ = cumulative.Save()
	}
	policyCfg := policy.NewConfig(fs)
	if policyCfg.Load() != nil {
		for _, zoneID := range []string{"north", "south"} {
			if err := policyCfg.SetTarget(policy.Target{
				Zone:      zoneID,
				TargetMPA: 0.40,
				MinMPA:    0.25,
				MaxMPA:    0.60,
				Strategy:  "constant-pressure",
			}); err != nil {
				return nil, err
			}
		}
	}
	evaluator := policy.NewEvaluator(windows, policyCfg)
	decider := policy.NewDecider(policyCfg)
	receipts := store.NewExecutionLog(fs)
	exec := dispatch.NewExecutor(receipts, recorder)
	guard := hammer.NewGuard()
	valveExec := valve.NewExecutor(receipts, fs)
	snapshots := store.NewSnapshotService(fs)
	queue := dispatch.NewOrderQueue()
	server := New(fs, registry, st, mapper, sampler, windows, agg, ledger, recorder, exec, guard, valveExec, cumulative, policyCfg, evaluator, decider, snapshots, queue)
	server.group1 = group1
	server.group2 = group2
	server.valveUp = valveUp
	server.valveDown = valveDown
	server.valveIso = valveIso
	server.meter1 = pressure.NewMeter("m-1", "north")
	server.tank1 = tank
	server.receipts = receipts
	if err := server.restore(); err != nil {
		return nil, err
	}
	_, _ = snapshots.Rotate()
	return server, nil
}

func (s *Server) restore() error {
	for _, group := range []*pump.Group{s.group1, s.group2} {
		loaded, err := pump.LoadGroup(s.fs, group.ID)
		if err == nil {
			if err := group.Restore(loaded.Record()); err != nil {
				return err
			}
		}
	}
	for _, v := range []*valve.Valve{s.valveUp, s.valveDown, s.valveIso} {
		var record store.ValveRecord
		if err := s.fs.ReadJSON("valves/"+v.ID+".json", &record); err == nil {
			if err := v.Restore(record); err != nil {
				return err
			}
		}
	}
	if state, ok, err := s.snapshots.Latest(); err == nil && ok {
		if len(state.ZoneMap) > 0 {
			s.mapper.Restore(state.ZoneMap)
		}
	}
	return nil
}
