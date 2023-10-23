package setup

import (
	"errors"
	daLogger "github.com/FatProteins/master-thesis-code/logger"
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
	"time"
)

var logger = daLogger.NewLogger("setup")

type FaultConfig struct {
	EducationMode              bool   `yaml:"education-mode"`
	UnixToDaDomainSocketPath   string `yaml:"unix-to-da-domain-socket-path"`
	UnixFromDaDomainSocketPath string `yaml:"unix-from-da-domain-socket-path"`
	FaultsEnabled              bool   `yaml:"faults-enabled"`
	Actions                    struct {
		Noop struct {
		} `yaml:"noop"`
		Halt struct {
			MaxDuration int `yaml:"max-duration"`
		} `yaml:"halt"`
		Pause struct {
			PauseCommand string `yaml:"pause-command"`
		} `yaml:"pause"`
		Stop struct {
			StopCommand string `yaml:"stop-command"`
		} `yaml:"stop"`
		ResendLastMessage struct {
			Probability float64 `yaml:"probability"`
			MaxDuration int     `yaml:"max-duration"`
		} `yaml:"resend-last-message"`
		Continue struct {
			ContinueCommand string `yaml:"continue-command"`
		} `yaml:"continue"`
		Restart struct {
			RestartCommand string `yaml:"restart-command"`
		} `yaml:"restart"`
	} `yaml:"actions"`
}

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

	if len(config.Actions.Pause.PauseCommand) == 0 {
		return errors.Join(baseErr, errors.New("pause command is empty"))
	}

	if len(config.Actions.Continue.ContinueCommand) == 0 {
		return errors.Join(baseErr, errors.New("unpause command is empty"))
	}

	if len(config.Actions.Stop.StopCommand) == 0 {
		return errors.Join(baseErr, errors.New("stop command is empty"))
	}

	if len(config.Actions.Restart.RestartCommand) == 0 {
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
	actions        map[protocol.ActionType]FaultAction
	config         FaultConfig
	stepByStepMode bool
}

func NewActionPicker(config FaultConfig) *ActionPicker {
	pauseCmd, pauseArgs := splitCommand(config.Actions.Pause.PauseCommand)
	continueCmd, continueArgs := splitCommand(config.Actions.Continue.ContinueCommand)
	stopCmd, stopArgs := splitCommand(config.Actions.Stop.StopCommand)
	restartCmd, restartArgs := splitCommand(config.Actions.Restart.RestartCommand)
	actions := map[protocol.ActionType]FaultAction{
		protocol.ActionType_NOOP_ACTION_TYPE:                &NoopAction{},
		protocol.ActionType_HALT_ACTION_TYPE:                &HaltAction{config},
		protocol.ActionType_PAUSE_ACTION_TYPE:               &PauseAction{config, pauseCmd, pauseArgs},
		protocol.ActionType_STOP_ACTION_TYPE:                &StopAction{config, stopCmd, stopArgs},
		protocol.ActionType_RESEND_LAST_MESSAGE_ACTION_TYPE: &ResendLastMessageAction{},
		protocol.ActionType_CONTINUE_ACTION_TYPE:            &ContinueAction{config, continueCmd, continueArgs},
		protocol.ActionType_RESTART_ACTION_TYPE:             &RestartAction{config, restartCmd, restartArgs},
	}
	return &ActionPicker{actions: actions, config: config}
}

func (actionPicker *ActionPicker) DetermineAction() FaultAction {
	if actionPicker.config.EducationMode && actionPicker.stepByStepMode {
		return actionPicker.actions[protocol.ActionType_PAUSE_ACTION_TYPE]
	}

	return actionPicker.actions[protocol.ActionType_NOOP_ACTION_TYPE]
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
	time.Sleep(time.Duration(action.config.Actions.Halt.MaxDuration) * time.Millisecond)
}

type PauseAction struct {
	config FaultConfig

	pauseCmd  string
	pauseArgs []string
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
	err := exec.Command(action.pauseCmd, action.pauseArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute pause command")
	}
}

type ContinueAction struct {
	config FaultConfig

	continueCmd  string
	continueArgs []string
}

func (action *ContinueAction) Name() string {
	return "Continue"
}

func (action *ContinueAction) Perform(func()) {
	err := exec.Command(action.continueCmd, action.continueArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute continue command")
		return
	}
}

func (action *ContinueAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

type StopAction struct {
	config FaultConfig

	stopCmd  string
	stopArgs []string
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
	logger.Info("Stopping container with command %s", action.stopCmd)
	err := exec.Command(action.stopCmd, action.stopArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute stop command")
		return
	}

	logger.Info("Resetting connection...")
	resetConnFunc()
}

type RestartAction struct {
	config FaultConfig

	restartCmd  string
	restartArgs []string
}

func (action *RestartAction) Perform(func()) {
	logger.Info("Restarting container with command %s", action.restartCmd)
	logger.Info("Restarting container with args %s", action.restartArgs)
	err := exec.Command(action.restartCmd, action.restartArgs...).Run()
	if err != nil {
		logger.ErrorErr(err, "Failed to execute restart command")
		return
	}

	logger.Info("Restarted container")
}

func (action *RestartAction) GenerateResponse(response *protocol.Message) error {
	response.Reset()
	response.MessageType = protocol.MessageType_DA_RESPONSE
	response.MessageObject = &anypb.Any{}
	err := response.MessageObject.MarshalFrom(response)
	if err != nil {
		return err
	}

	return nil
}

func (action *RestartAction) Name() string {
	return "Restart"
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
