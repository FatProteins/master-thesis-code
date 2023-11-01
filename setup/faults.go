package setup

import (
	"errors"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat/distuv"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

var logger = daLogger.NewLogger("setup")

type FaultConfig struct {
	UnixToDaDomainSocketPath   string `yaml:"unix-to-da-domain-socket-path"`
	UnixFromDaDomainSocketPath string `yaml:"unix-from-da-domain-socket-path"`
	FaultsEnabled              bool   `yaml:"faults-enabled"`
	Actions                    struct {
		Noop struct {
			Probability float64 `yaml:"probability"`
		} `yaml:"noop"`
		Halt struct {
			Probability float64 `yaml:"probability"`
			HaltCommand string  `yaml:"halt-command"`
		} `yaml:"halt"`
		Unhalt struct {
			Probability   float64 `yaml:"probability"`
			UnhaltCommand string  `yaml:"unhalt-command"`
		}
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
	Perform(resetConnFunc func())
	Name() string
	GenerateResponse(*protocol.Message) error
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
	if len(config.UnixToDaDomainSocketPath) == 0 {
		return errors.Join(baseErr, errors.New("unix to DA domain socket path is empty"))
	}

	if len(config.UnixFromDaDomainSocketPath) == 0 {
		return errors.Join(baseErr, errors.New("unix from DA domain socket path is empty"))
	}

	if len(config.Actions.Halt.HaltCommand) == 0 {
		return errors.Join(baseErr, errors.New("halt command is empty"))
	}

	if len(config.Actions.Unhalt.UnhaltCommand) == 0 {
		return errors.Join(baseErr, errors.New("unhalt command is empty"))
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
	actions          map[protocol.ActionType]FaultAction
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

	haltCmd, haltArgs := splitCommand(config.Actions.Halt.HaltCommand)
	unhaltCmd, unhaltArgs := splitCommand(config.Actions.Unhalt.UnhaltCommand)
	pauseCmd, pauseArgs := splitCommand(config.Actions.Pause.PauseCommand)
	continueCmd, continueArgs := splitCommand(config.Actions.Pause.ContinueCommand)
	stopCmd, stopArgs := splitCommand(config.Actions.Stop.StopCommand)
	restartCmd, restartArgs := splitCommand(config.Actions.Stop.RestartCommand)
	actions := map[protocol.ActionType]FaultAction{
		protocol.ActionType_NOOP_ACTION_TYPE:                &NoopAction{},
		protocol.ActionType_HALT_ACTION_TYPE:                &HaltAction{config, haltCmd, haltArgs},
		protocol.ActionType_UNHALT_ACTION_TYPE:              &UnhaltAction{config, unhaltCmd, unhaltArgs},
		protocol.ActionType_PAUSE_ACTION_TYPE:               &PauseAction{config, pauseCmd, pauseArgs, continueCmd, continueArgs},
		protocol.ActionType_STOP_ACTION_TYPE:                &StopAction{config, stopCmd, stopArgs, restartCmd, restartArgs},
		protocol.ActionType_RESEND_LAST_MESSAGE_ACTION_TYPE: &ResendLastMessageAction{},
	}
	return &ActionPicker{cumProbabilities: cumSum, actions: actions}
}

func (actionPicker *ActionPicker) DetermineAction() FaultAction {
	val := distuv.UnitUniform.Rand() * actionPicker.cumProbabilities[len(actionPicker.cumProbabilities)-1]
	actionIdx := sort.Search(len(actionPicker.cumProbabilities), func(i int) bool { return actionPicker.cumProbabilities[i] > val })
	action := actionPicker.actions[protocol.ActionType(actionIdx)]
	logger.Debug("Picking action '%s'", action.Name())
	return action
}

func (actionPicker *ActionPicker) GetAction(actionType protocol.ActionType) FaultAction {
	return actionPicker.actions[actionType]
}

type NoopAction struct {
}

func (action *NoopAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *NoopAction) Perform(func()) {
	// Do nothing
}

func (action *NoopAction) Name() string {
	return "Noop"
}

type HaltAction struct {
	config FaultConfig

	haltCmd  string
	haltArgs []string
}

func (action *HaltAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *HaltAction) Name() string {
	return "Halt"
}

func (action *HaltAction) Perform(func()) {
	err := exec.Command(action.haltCmd, action.haltArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute halt command")
		return
	}
}

type UnhaltAction struct {
	config FaultConfig

	unhaltCmd  string
	unhaltArgs []string
}

func (action *UnhaltAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *UnhaltAction) Name() string {
	return "Unhalt"
}

func (action *UnhaltAction) Perform(func()) {
	err := exec.Command(action.unhaltCmd, action.unhaltArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute unhalt command")
		return
	}
}

type PauseAction struct {
	config FaultConfig

	pauseCmd  string
	pauseArgs []string

	continueCmd  string
	continueArgs []string
}

func (action *PauseAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *PauseAction) Name() string {
	return "Pause"
}

func (action *PauseAction) Perform(func()) {
	pauseConfig := action.config.Actions.Pause
	err := exec.Command(action.pauseCmd, action.pauseArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute pause command")
		return
	}

	time.Sleep(time.Duration(pauseConfig.MaxDuration) * time.Millisecond)
	err = exec.Command(action.continueCmd, action.continueArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute continue command")
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

func (action *StopAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *StopAction) Name() string {
	return "Stop"
}

func (action *StopAction) Perform(resetConnFunc func()) {
	stopConfig := &action.config.Actions.Stop
	logger.Info("Stopping container with command %s", action.stopCmd)
	err := exec.Command(action.stopCmd, action.stopArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute stop command")
		return
	}

	logger.Info("Resetting connection...")
	resetConnFunc()

	logger.Info("Waiting after stop...")
	time.Sleep(time.Duration(stopConfig.MaxDuration) * time.Millisecond)
	logger.Info("Restarting container with command %s", action.restartCmd)
	logger.Info("Restarting container with args %s", action.restartArgs)
	err = exec.Command(action.restartCmd, action.restartArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute restart command")
		return
	}

	logger.Info("Restarted container")
}

type ResendLastMessageAction struct {
}

func (action *ResendLastMessageAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *ResendLastMessageAction) Name() string {
	return "ResendLastMessage"
}

func (action *ResendLastMessageAction) Perform(func()) {

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
