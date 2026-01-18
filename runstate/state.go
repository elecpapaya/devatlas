package runstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	LastRunAt time.Time `json:"last_run_at"`
}

func Load(path string) (*State, error) {
	if path == "" {
		return nil, errors.New("runstate: path is required")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func Save(path string, state *State) error {
	if path == "" {
		return errors.New("runstate: path is required")
	}
	if state == nil {
		return errors.New("runstate: state is nil")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}
