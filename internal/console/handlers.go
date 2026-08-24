package console

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"waternet/internal/audit"
	"waternet/internal/dispatch"
	"waternet/internal/hammer"
	"waternet/internal/ns"
	"waternet/internal/policy"
	"waternet/internal/pressure"
	"waternet/internal/pump"
	"waternet/internal/station"
	"waternet/internal/store"
	"waternet/internal/valve"
)

func (s *Server) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	var out []map[string]any
	for _, n := range s.registry.All() {
		out = append(out, map[string]any{"id": n.ID, "name": n.Name, "zones": n.ZoneIDs(), "zone_count": n.ZoneCount()})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleZone(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "zoneID")
	n, err := s.registry.Get("downtown")
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	zone, err := n.Zone(zoneID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	spec := n.Spec(zoneID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id": zone.ID, "name": zone.Name,
		"spec":            spec,
		"within_envelope": ns.WithinEnvelope(spec, 0.40),
	})
}

func (s *Server) handleStation(w http.ResponseWriter, r *http.Request) {
	state, err := s.station.CurrentState(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	record := s.station.Record()
	writeJSON(w, http.StatusOK, map[string]any{
		"station": record,
		"state":   state,
		"pumps":   s.station.PumpIDs(),
		"valves":  s.station.ValveIDs(),
	})
}

func (s *Server) handleTankFill(w http.ResponseWriter, r *http.Request) {
	tankID := chi.URLParam(r, "tankID")
	tank, err := s.station.Tank(tankID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		Delta float64 `json:"delta"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tank.Fill(body.Delta); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"level": tank.LevelNow(), "remaining": tank.Remaining(), "overflowing": tank.Overflowing(),
	})
}

func (s *Server) handlePumps(w http.ResponseWriter, r *http.Request) {
	var out []map[string]any
	for _, group := range []*pump.Group{s.group1, s.group2} {
		out = append(out, map[string]any{
			"id": group.ID, "name": group.Name, "active_unit": group.ActiveUnit,
			"running": group.RunningUnits(), "healthy": group.HealthyUnits(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePump(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	units := make([]map[string]any, 0, len(group.Units))
	activeDetail := ""
	if group.ActiveUnit != "" {
		if active, err := group.Unit(group.ActiveUnit); err == nil {
			activeDetail = active.Name
		}
	}
	for _, unit := range group.Units {
		units = append(units, map[string]any{
			"id": unit.ID, "name": unit.Name, "status": unit.Status,
			"locked": unit.Locked, "vibration": unit.Vibration,
			"is_primary": group.IsPrimary(unit.ID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": group.ID, "station": group.StationID, "active_unit": group.ActiveUnit,
		"active_unit_name": activeDetail, "discharge_mpa": group.DischargeMPA, "target_mpa": group.TargetMPA, "units": units,
	})
}

func (s *Server) handlePumpMaintenance(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := station.LockPumpForMaintenance(r.Context(), s.station, group, body.UnitID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok(w, "unit locked for maintenance")
}

func (s *Server) handlePumpRelease(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := station.ReleasePump(r.Context(), s.station, group, body.UnitID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	locked, _ := group.IsLocked(body.UnitID)
	writeJSON(w, http.StatusOK, map[string]any{"released": true, "unit_id": body.UnitID, "is_locked": locked})
}

func (s *Server) handlePumpStart(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.StartUnit(body.UnitID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "unit started")
}

func (s *Server) handlePumpStop(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.StopUnit(body.UnitID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "unit stopped")
}

func (s *Server) handlePumpDischarge(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		MPA float64 `json:"mpa"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.SetDischarge(body.MPA); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "discharge updated")
}

func (s *Server) handlePumpTarget(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		MPA float64 `json:"mpa"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.SetTarget(body.MPA); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "target updated")
}

func (s *Server) handlePumpStatus(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.SetStatus(body.UnitID, pump.Status(body.Status)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "status updated")
}

func (s *Server) handlePumpVibration(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string  `json:"unit_id"`
		Value  float64 `json:"value"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := group.RecordVibration(body.UnitID, body.Value); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"over_limit": group.VibrationOverLimit()})
}

func (s *Server) handlePumpPreflight(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := pump.Preflight(r.Context(), group); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"allowed": false, "reason": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"allowed": true})
}

func (s *Server) handlePumpTrip(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		UnitID string `json:"unit_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := pump.Trip(r.Context(), group, body.UnitID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = group.Save()
	ok(w, "unit tripped")
}

func (s *Server) handleFailover(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	active, err := pump.Failover(r.Context(), group, s.station)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active_unit": active})
}

func (s *Server) handlePumpCommandsBatch(w http.ResponseWriter, r *http.Request) {
	group, err := s.pumpByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		Commands []pump.Command `json:"commands"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	count, err := pump.ApplyCommandList(r.Context(), group, body.Commands)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": count})
}

func (s *Server) handleValves(w http.ResponseWriter, r *http.Request) {
	var out []valve.Status
	for _, v := range []*valve.Valve{s.valveUp, s.valveDown, s.valveIso} {
		out = append(out, valve.StatusOf(v))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleValve(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"valve":      valve.StatusOf(v),
		"fully_open": valve.FullyOpen(v), "fully_closed": valve.FullyClosed(v),
	})
}

func (s *Server) handleValveZone(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		Zone string `json:"zone"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := valve.SwitchZone(r.Context(), v, body.Zone, s.mapper); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"zone": valve.IsolationZone(v), "attribution": s.mapper.ZoneFor(v.ID),
	})
}

func (s *Server) handleValveClose(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	command := valve.NewCommand(v.ID, valve.ActionClose)
	if err := valve.Execute(r.Context(), v, command, s.guard, s.valveExec); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	plan := hammer.PlanForStrokes(s.mapper.ZoneFor(v.ID), s.valveUp.ID, v.ID)
	closureAt := time.Now().UTC()
	if err := hammer.RecordClosurePeak(r.Context(), s.windows, plan, closureAt, v.PositionNow()*0.6); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	peak, _ := hammer.PeakInWindow(r.Context(), s.windows, plan, closureAt)
	s.guard.Release(v.ID)
	writeJSON(w, http.StatusOK, map[string]any{"closed": true, "peak_mpa": peak, "window_covered": hammer.ClosurePeakWindow(plan, closureAt).Contains(closureAt)})
}

func (s *Server) handleValveCloseFully(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := s.valveExec.CloseFully(r.Context(), v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, "valve fully closed")
}

func (s *Server) handleValveOpen(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := s.valveExec.OpenFully(r.Context(), v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, "valve opened")
}

func (s *Server) handleValveProtection(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	executions, _ := s.exec.Receipts(0)
	closeReceipts := 0
	for _, record := range executions {
		if record.Target == v.ID && record.Action == string(valve.ActionClose) {
			closeReceipts++
		}
	}
	countByTarget, _ := s.receipts.CountByTarget(v.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"engaged": s.guard.IsEngaged(v.ID), "close_count": s.guard.CloseCount(v.ID),
		"receipt_close_count": closeReceipts, "count_by_target": countByTarget,
	})
}

func (s *Server) handleValveDetach(w http.ResponseWriter, r *http.Request) {
	v, err := s.valveByID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	s.mapper.DetachZone(v.ID)
	writeJSON(w, http.StatusOK, map[string]any{"valve_id": v.ID, "attribution": s.mapper.ZoneFor(v.ID)})
}

func (s *Server) handleValveProtect(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UpstreamID   string `json:"upstream_id"`
		DownstreamID string `json:"downstream_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	upstream, err := s.valveByIDValue(body.UpstreamID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	downstream, err := s.valveByIDValue(body.DownstreamID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	zone := s.mapper.ZoneFor(body.DownstreamID)
	plan := hammer.PlanForStrokes(zone, body.UpstreamID, body.DownstreamID)
	valves := map[string]*valve.Valve{body.UpstreamID: upstream, body.DownstreamID: downstream}
	steps, err := hammer.Protect(r.Context(), s.valveExec, plan, valves)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"steps": steps, "order": plan.OrderedValveIDs()})
}

func (s *Server) handleSample(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ValveID string  `json:"valve_id"`
		MPA     float64 `json:"mpa"`
		Flow    float64 `json:"flow"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	reading, err := s.sampler.Sample(r.Context(), body.ValveID, body.MPA, body.Flow, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reading)
}

func (s *Server) handleSampleBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ValveID  string             `json:"valve_id"`
		Readings []pressure.Reading `json:"readings"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	count, err := s.sampler.SampleBatch(r.Context(), body.ValveID, body.Readings)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": count})
}

func (s *Server) handlePressureWindows(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	slots, err := s.agg.ZoneSlots(zone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []map[string]any
	for _, slot := range slots {
		peak, _ := s.agg.Peak(zone, slot)
		count, _ := s.agg.Count(zone, slot)
		out = append(out, map[string]any{"slot": slot, "peak_mpa": peak, "samples": count})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePressureAdjust(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GroupID string  `json:"group_id"`
		Target  float64 `json:"target"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := s.station.Group(body.GroupID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := pressure.Adjust(r.Context(), group, body.Target); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"adjusted": false, "reason": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"adjusted": true, "target": body.Target})
}

func (s *Server) handlePressureAdjustBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Groups  []string  `json:"groups"`
		Targets []float64 `json:"targets"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	groups := make([]*pump.Group, 0, len(body.Groups))
	for _, id := range body.Groups {
		group, err := s.station.Group(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		groups = append(groups, group)
	}
	if err := pressure.AdjustBatch(r.Context(), groups, body.Targets); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	ok(w, "batch adjusted")
}

func (s *Server) handleDispatchSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session := s.Session(body.Name)
	writeJSON(w, http.StatusOK, map[string]any{"session_id": session.ID()})
}

func (s *Server) handleDispatchCommand(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string  `json:"session_id"`
		GroupID   string  `json:"group_id"`
		UnitID    string  `json:"unit_id"`
		Action    string  `json:"action"`
		TargetMPA float64 `json:"target_mpa"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := s.station.Group(body.GroupID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	command := pump.NewCommand(body.GroupID, body.UnitID, body.Action)
	command.TargetMPA = body.TargetMPA
	if err := s.Session(body.SessionID).Send(r.Context(), command, group); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"command_id": command.ID})
}

func (s *Server) handleDispatchQueue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GroupID   string  `json:"group_id"`
		UnitID    string  `json:"unit_id"`
		Action    string  `json:"action"`
		TargetMPA float64 `json:"target_mpa"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	command := pump.NewCommand(body.GroupID, body.UnitID, body.Action)
	command.TargetMPA = body.TargetMPA
	seq := s.NextSeq()
	if err := station.DispatchCommand(s.queue, seq, command); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issue_seq": seq})
}

func (s *Server) handleDispatchQueueStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"pending": s.queue.Len()})
}

func (s *Server) handleDispatchQueueReset(w http.ResponseWriter, r *http.Request) {
	s.queue.Reset()
	ok(w, "queue reset")
}

func (s *Server) handleDispatchApply(w http.ResponseWriter, r *http.Request) {
	count, err := station.ApplyDispatch(r.Context(), s.station, s.queue)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": count})
}

func (s *Server) handlePolicyDecideBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Zone    string          `json:"zone"`
		GroupID string          `json:"group_id"`
		UnitID  string          `json:"unit_id"`
		Results []policy.Result `json:"results"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	commands, err := s.decider.DecideBatch(body.Zone, body.Results, body.GroupID, body.UnitID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"commands": commands})
}

func (s *Server) handlePolicyEvaluate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Zone string  `json:"zone"`
		MPA  float64 `json:"mpa"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	readings := []store.ReadingRecord{{Zone: body.Zone, MPA: body.MPA}}
	result, err := s.evaluator.Evaluate(body.Zone, readings)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handlePolicyList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.policyCfg.All())
}

func (s *Server) handlePolicySet(w http.ResponseWriter, r *http.Request) {
	var target policy.Target
	if err := decodeJSON(r, &target); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.policyCfg.SetTarget(target); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok(w, "policy published")
}

func (s *Server) handlePolicySave(w http.ResponseWriter, r *http.Request) {
	if err := s.policyCfg.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, "policy persisted")
}

func (s *Server) handleMeter(w http.ResponseWriter, r *http.Request) {
	meterID := chi.URLParam(r, "id")
	segment, err := s.cumulative.SegmentTotal(meterID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	lifetime, _ := s.cumulative.Total(meterID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id": meterID, "zone": s.meter1.Zone,
		"direction":     pressure.DirectionOf(s.cumulative, meterID),
		"segment_total": segment, "lifetime_total": lifetime,
	})
}

func (s *Server) handleMeterDirection(w http.ResponseWriter, r *http.Request) {
	meterID := chi.URLParam(r, "id")
	var body struct {
		Direction string `json:"direction"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := pressure.SwitchDirection(r.Context(), s.cumulative, meterID, body.Direction); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.cumulative.Save()
	writeJSON(w, http.StatusOK, map[string]any{"direction": pressure.DirectionOf(s.cumulative, meterID)})
}

func (s *Server) handleMeterFlow(w http.ResponseWriter, r *http.Request) {
	meterID := chi.URLParam(r, "id")
	var body struct {
		Delta float64 `json:"delta"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.meter1.AddFlow(r.Context(), s.cumulative, body.Delta); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	segment, _ := s.meter1.SegmentTotal(s.cumulative)
	lifetime, _ := s.meter1.LifetimeTotal(s.cumulative)
	_ = s.cumulative.Save()
	writeJSON(w, http.StatusOK, map[string]any{"meter_id": meterID, "segment_total": segment, "lifetime_total": lifetime})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	events, err := s.recorder.Recent(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleAuditSummary(w http.ResponseWriter, r *http.Request) {
	events, err := s.recorder.Recent(0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, audit.Summarize(events, 10))
}

func (s *Server) handleAuditRaw(w http.ResponseWriter, r *http.Request) {
	lines, err := s.fs.ReadLines("audit.jsonl")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	events := make([]audit.Event, 0, len(lines))
	for _, line := range lines {
		event, err := audit.DecodeEvent(line)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		events = append(events, event)
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleQuota(w http.ResponseWriter, r *http.Request) {
	zone := chi.URLParam(r, "zone")
	writeJSON(w, http.StatusOK, map[string]any{
		"status": s.ledger.Status(zone), "remaining": s.ledger.Remaining(zone),
	})
}

func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	records, err := s.exec.Receipts(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	summary := dispatch.ReceiptSummary(records)
	writeJSON(w, http.StatusOK, map[string]any{
		"records": records, "summary": summary, "actions": dispatch.SortedActions(summary),
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	fingerprint, err := s.fs.StateFingerprint()
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	persisted, _ := s.fs.LoadState()
	cached, _ := s.station.CachedPumpStatuses(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"state": s.StateRecord(), "persisted": persisted, "cached_pumps": cached, "fingerprint": fingerprint,
	})
}

func (s *Server) handleStoreFiles(w http.ResponseWriter, r *http.Request) {
	names, err := s.fs.ListNames(".")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, names)
}

func (s *Server) pumpByID(r *http.Request) (*pump.Group, error) {
	id := chi.URLParam(r, "id")
	if id == s.group1.ID {
		return s.group1, nil
	}
	if s.group2 != nil && id == s.group2.ID {
		return s.group2, nil
	}
	return nil, errors.New("pump group not found")
}

func (s *Server) valveByID(r *http.Request) (*valve.Valve, error) {
	id := chi.URLParam(r, "id")
	return s.valveByIDValue(id)
}

func (s *Server) valveByIDValue(id string) (*valve.Valve, error) {
	for _, v := range []*valve.Valve{s.valveUp, s.valveDown, s.valveIso} {
		if v != nil && v.ID == id {
			return v, nil
		}
	}
	return nil, errors.New("valve not found")
}
