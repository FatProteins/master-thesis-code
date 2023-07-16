package setup

import (
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat/distuv"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"sort"
	"time"
)

type FaultConfig struct {
	UnixDomainSocketPath string `yaml:"unix-domain-socket-path"`
	FaultsEnabled        bool   `yaml:"faults-enabled"`
	Actions              struct {
		Noop struct {
			Probability float64 `yaml:"probability"`
		} `yaml:"noop"`
		Halt struct {
			Probability float64 `yaml:"probability"`
			MaxDuration int     `yaml:"max-duration"`
		} `yaml:"halt"`
		Pause struct {
			Probability     float64 `yaml:"probability"`
			MaxDuration     int     `yaml:"max-duration"`
			PauseCommand    string  `yaml:"pause-command"`
			ContinueCommand string  `yaml:"continue-command"`
		} `yaml:"pause"`
		Stop struct {
			Probability    float64 `yaml:"probability"`
			MaxDuration    int     `yaml:"max-duration"`
			StopCommand    string  `yaml:"stop-command"`
			RestartCommand string  `yaml:"restart-command"`
		} `yaml:"stop"`
		ResendLastMessage struct {
			Probability float64 `yaml:"probability"`
			MaxDuration int     `yaml:"max-duration"`
		} `yaml:"resend-last-message"`
	} `yaml:"actions"`
}

const (
	noopAction int = iota
	haltAction
	pauseAction
	stopAction
	resendLastMessageAction
)

type FaultAction interface {
	Perform()
}

func ReadFaultConfig(path string) (FaultConfig, error) {
	var config FaultConfig
	content, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

type ActionPicker struct {
	cumProbabilities []float64
	actions          map[int]FaultAction
}

func NewActionPicker(config FaultConfig) *ActionPicker {
	probabilities := []float64{
		config.Actions.Noop.Probability,
		config.Actions.Halt.Probability,
		config.Actions.Pause.Probability,
		config.Actions.Stop.Probability,
		config.Actions.ResendLastMessage.Probability,
	}
	cumSum := make([]float64, 5, 5)
	floats.CumSum(cumSum, probabilities)

	actions := map[int]FaultAction{
		noopAction:              &NoopAction{},
		haltAction:              &HaltAction{config},
		pauseAction:             &PauseAction{config},
		stopAction:              &StopAction{config},
		resendLastMessageAction: &ResendLastMessageAction{},
	}
	return &ActionPicker{cumProbabilities: cumSum, actions: actions}
}

func (actionPicker *ActionPicker) DetermineAction() FaultAction {
	val := distuv.UnitUniform.Rand() * actionPicker.cumProbabilities[len(actionPicker.cumProbabilities)-1]
	actionIdx := sort.Search(len(actionPicker.cumProbabilities), func(i int) bool { return actionPicker.cumProbabilities[i] > val })
	return actionPicker.actions[actionIdx]
}

type NoopAction struct {
}

func (action *NoopAction) Perform() {
	// Do nothing
}

type HaltAction struct {
	config FaultConfig
}

func (action *HaltAction) Perform() {
	// TODO: Introduce randomness
	time.Sleep(time.Duration(action.config.Actions.Halt.MaxDuration) * time.Millisecond)
}

type PauseAction struct {
	config FaultConfig
}

func (action *PauseAction) Perform() {
	pauseConfig := action.config.Actions.Pause
	err := exec.Command(pauseConfig.PauseCommand).Run()
	if err != nil {
		return
	}

	time.Sleep(time.Duration(pauseConfig.MaxDuration) * time.Millisecond)
	err = exec.Command(pauseConfig.ContinueCommand).Run()
	if err != nil {
		return
	}
}

type StopAction struct {
	config FaultConfig
}

func (action *StopAction) Perform() {
	stopConfig := &action.config.Actions.Stop
	err := exec.Command(stopConfig.StopCommand).Run()
	if err != nil {
		return
	}

	time.Sleep(time.Duration(stopConfig.MaxDuration) * time.Millisecond)
	err = exec.Command(stopConfig.RestartCommand).Run()
	if err != nil {
		return
	}
}

type ResendLastMessageAction struct {
}

func (action *ResendLastMessageAction) Perform() {

}
