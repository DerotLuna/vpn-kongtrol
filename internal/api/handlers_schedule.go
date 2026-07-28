package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// GET /api/v1/scheduler/rules
func (s *Server) handleListScheduleRules(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]scheduleRuleDTO, 0, len(cfg.Monitor.Scheduler.Rules))
	for _, rule := range cfg.Monitor.Scheduler.Rules {
		out = append(out, scheduleRuleDTO{
			Name:     rule.Name,
			Profiles: rule.Profiles,
			Weekdays: rule.Weekdays,
			Start:    rule.Start,
			End:      rule.End,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /api/v1/scheduler/rules
func (s *Server) handleCreateScheduleRule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "rule name is required")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, req.Name) {
			writeError(w, http.StatusConflict, "scheduler rule already exists")
			return
		}
	}
	if err := s.saveScheduleRule(cfg, cfgPath, -1, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "rule": req.Name})
}

// PUT /api/v1/scheduler/rules/{name}
func (s *Server) handleUpdateScheduleRule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req scheduleRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := -1
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeError(w, http.StatusNotFound, "scheduler rule not found")
		return
	}
	if err := s.saveScheduleRule(cfg, cfgPath, idx, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "rule": name})
}

func (s *Server) saveScheduleRule(cfg *config.Config, cfgPath string, idx int, req scheduleRuleDTO) error {
	rule := config.ScheduleRule{
		Name:     strings.TrimSpace(req.Name),
		Profiles: req.Profiles,
		Weekdays: req.Weekdays,
		Start:    strings.TrimSpace(req.Start),
		End:      strings.TrimSpace(req.End),
	}

	trial := *cfg
	trialRules := make([]config.ScheduleRule, len(cfg.Monitor.Scheduler.Rules))
	copy(trialRules, cfg.Monitor.Scheduler.Rules)
	if idx >= 0 {
		trialRules[idx] = rule
	} else {
		trialRules = append(trialRules, rule)
	}
	trial.Monitor.Scheduler.Rules = trialRules
	if err := config.Validate(&trial); err != nil {
		return err
	}

	cfg.Monitor.Scheduler.Rules = trialRules
	return s.saveRuntimeConfig(cfgPath, cfg)
}

// DELETE /api/v1/scheduler/rules/{name}
func (s *Server) handleDeleteScheduleRule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := -1
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeError(w, http.StatusNotFound, "scheduler rule not found")
		return
	}
	cfg.Monitor.Scheduler.Rules = append(cfg.Monitor.Scheduler.Rules[:idx], cfg.Monitor.Scheduler.Rules[idx+1:]...)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "rule": name})
}
