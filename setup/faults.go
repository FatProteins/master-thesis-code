package setup

import (
	"errors"
	"fmt"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat/distuv"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"sort"
	"strings"
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

	err = config.verifyConfig()
	if err != nil {
		return config, err
	}

	return config, nil
}

func (config *FaultConfig) verifyConfig() error {
	baseErr := errors.New("config error")
	if len(config.UnixDomainSocketPath) == 0 {
		return errors.Join(baseErr, errors.New("unix domain socket path is empty"))
	}

	if len(config.Actions.Pause.PauseCommand) == 0 {
		return errors.Join(baseErr, errors.New("pause command is empty"))
	}

	if len(config.Actions.Pause.ContinueCommand) == 0 {
		return errors.Join(baseErr, errors.New("unpause command is empty"))
	}

	if len(config.Actions.Stop.StopCommand) == 0 {
		return errors.Join(baseErr, errors.New("stop command is empty"))
	}

	if len(config.Actions.Stop.RestartCommand) == 0 {
		return errors.Join(baseErr, errors.New("restart command is empty"))
	}

	return nil
}

func (config *FaultConfig) String() (string, error) {
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
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

	pauseCmd, pauseArgs := splitCommand(config.Actions.Pause.PauseCommand)
	continueCmd, continueArgs := splitCommand(config.Actions.Pause.ContinueCommand)
	stopCmd, stopArgs := splitCommand(config.Actions.Stop.StopCommand)
	restartCmd, restartArgs := splitCommand(config.Actions.Stop.RestartCommand)
	actions := map[int]FaultAction{
		noopAction:              &NoopAction{},
		haltAction:              &HaltAction{config},
		pauseAction:             &PauseAction{config, pauseCmd, pauseArgs, continueCmd, continueArgs},
		stopAction:              &StopAction{config, stopCmd, stopArgs, restartCmd, restartArgs},
		resendLastMessageAction: &ResendLastMessageAction{},
	}
	return &ActionPicker{cumProbabilities: cumSum, actions: actions}
}

func (actionPicker *ActionPicker) DetermineAction() FaultAction {
	val := distuv.UnitUniform.Rand() * actionPicker.cumProbabilities[len(actionPicker.cumProbabilities)-1]
	actionIdx := sort.Search(len(actionPicker.cumProbabilities), func(i int) bool { return actionPicker.cumProbabilities[i] > val })
	fmt.Printf("Picking action '%d'\n", actionIdx)
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

	pauseCmd  string
	pauseArgs []string

	continueCmd  string
	continueArgs []string
}

func (action *PauseAction) Perform() {
	pauseConfig := action.config.Actions.Pause
	err := exec.Command(action.pauseCmd, action.pauseArgs...).Run()
	if err != nil {
		fmt.Printf("Failed to execute pause command: %s", err.Error())
		return
	}

	time.Sleep(time.Duration(pauseConfig.MaxDuration) * time.Millisecond)
	err = exec.Command(action.continueCmd, action.continueArgs...).Run()
	if err != nil {
		fmt.Printf("Failed to execute continue command: %s", err.Error())
		return
	}
}

type StopAction struct {
	config FaultConfig

	stopCmd  string
	stopArgs []string

	restartCmd  string
	restartArgs []string
}

func (action *StopAction) Perform() {
	stopConfig := &action.config.Actions.Stop
	err := exec.Command(action.stopCmd, action.stopArgs...).Run()
	if err != nil {
		fmt.Printf("Failed to execute stop command: %s", err.Error())
		return
	}

	time.Sleep(time.Duration(stopConfig.MaxDuration) * time.Millisecond)
	err = exec.Command(action.restartCmd, action.restartArgs...).Run()
	if err != nil {
		fmt.Printf("Failed to execute restart command: %s", err.Error())
		return
	}
}

type ResendLastMessageAction struct {
}

func (action *ResendLastMessageAction) Perform() {

}

func splitCommand(command string) (string, []string) {
	cmdSplit := strings.Split(command, " ")
	cmd := cmdSplit[0]
	var args []string
	if len(cmdSplit) > 1 {
		args = cmdSplit[1:]
	} else {
		args = []string{}
	}

	return cmd, args
}
