package main

import (
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/golang/protobuf/proto"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	kafkaconfig "github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type configuration struct {
	LogFormat    string
	LogVerbosity string
	Ack          bool
}

var (
	cfg              = defaultConfig()
	kafkaConfig      = kafkaconfig.DefaultConsumer()
	deployments      = make(chan deployment.Event, 64)
	messages         = make(chan error, 64)
	app              = tview.NewApplication()
	deploymentWidget = tview.NewTable()
	messageWidget    = tview.NewTextView()
	kafkaLogger      = logging.NewChanLogger()
)

func defaultConfig() configuration {
	return configuration{
		LogFormat:    "text",
		LogVerbosity: "trace",
		Ack:          false,
	}
}

func init() {
	flag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format, either 'json' or 'text'.")
	flag.StringVar(&cfg.LogVerbosity, "log-verbosity", cfg.LogVerbosity, "Logging verbosity level.")
	flag.BoolVar(&cfg.Ack, "ack", cfg.Ack, "Acknowledge messages in Kafka queue, i.e. store consumer group position")

	kafkaconfig.SetupFlags(&kafkaConfig)
}

func filtered(event *deployment.Event) bool {
	if event.GetRolloutStatus() != deployment.RolloutStatus_complete {
		return true
	}
	return false
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var err error

	flag.Parse()

	kafkaconfig.SetLogger(kafkaLogger)

	err = logging.Apply(log.StandardLogger(), cfg.LogVerbosity, cfg.LogFormat)
	if err != nil {
		return err
	}

	kafka, err := consumer.New(kafkaConfig)
	if err != nil {
		return err
	}

	process := func() (*deployment.Event, error) {
		select {
		case msg, ok := <-kafka.Consume():
			if !ok {
				return nil, fmt.Errorf("kafka consumer has shut down")
			}

			if cfg.Ack {
				msg.Ack()
			}
			data := msg.M.Value

			event := &deployment.Event{}
			err = proto.Unmarshal(data, event)
			if err != nil {
				return nil, fmt.Errorf("unable to unmarshal incoming message: %s", err)
			}

			return event, nil
		}
	}

	go func() {
		for {
			event, err := process()
			if err == nil {
				if filtered(event) {
					continue
				}
				deployments <- *event
			} else {
				messages <- err
			}
		}
	}()

	log.SetOutput(messageWidget)

	return setupui()

}

func copytoui() {
	rows := 1
	for {
		select {
		case event := <-deployments:
			app.QueueUpdateDraw(func() {
				deploymentWidget.SetCell(rows, 0, tview.NewTableCell(event.GetApplication()))
				deploymentWidget.SetCellSimple(rows, 1, event.GetVersion())
				deploymentWidget.SetCellSimple(rows, 2, event.GetCluster())
				deploymentWidget.SetCellSimple(rows, 3, event.GetNamespace())
				deploymentWidget.SetCellSimple(rows, 4, event.GetTeam())
				deploymentWidget.SetCellSimple(rows, 5, event.GetPlatform().GetType().String())
				deploymentWidget.SetCellSimple(rows, 6, event.GetSource().String())
				deploymentWidget.SetCellSimple(rows, 7, event.GetCorrelationID())
				deploymentWidget.SetCellSimple(rows, 8, event.GetTimestampAsTime().String())
				rows++
			})
		case err := <-messages:
			app.QueueUpdateDraw(func() {
				messageWidget.Write([]byte(err.Error()))
				messageWidget.Write([]byte{'\n'})
			})
		case logLine := <-kafkaLogger.C:
			app.QueueUpdateDraw(func() {
				messageWidget.Write([]byte(logLine))
			})
		}
	}
}

func setupui() error {
	go copytoui()

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(deploymentWidget, 0, 100, true)
	flex.AddItem(messageWidget, 0, 20, true)
	flex.SetBackgroundColor(tcell.ColorBlack)

	messageWidget.
		SetBackgroundColor(tcell.ColorDefault).
		SetTitle("Messages").
		SetBorder(true).
		SetTitleColor(tcell.ColorGreen)

	deploymentWidget.
		SetCellSimple(1, 0, "Waiting for first deployment...").
		SetFixed(1, 0).
		SetBackgroundColor(tcell.ColorDefault).
		SetTitle("Deployments").
		SetBorder(true).
		SetTitleColor(tcell.ColorGreen)

	for column, label := range []string{
		"Application",
		"Version",
		"Cluster",
		"Namespace",
		"Team",
		"Platform",
		"Source",
		"ID",
		"Time",
	} {
		cell := tview.NewTableCell(label).SetTextColor(tcell.ColorLightGreen)
		deploymentWidget.SetCell(0, column, cell)
	}

	app.SetRoot(flex, true)

	capture := func(e *tcell.EventKey) *tcell.EventKey {
		if e.Name() == "Tab" {
			app.QueueUpdateDraw(func() {
				if deploymentWidget.HasFocus() {
					app.SetFocus(messageWidget)
				} else {
					app.SetFocus(deploymentWidget)
				}
			})
			return nil
		}
		return e
	}

	deploymentWidget.SetInputCapture(capture)
	messageWidget.SetInputCapture(capture)

	app.SetFocus(deploymentWidget)

	return app.Run()
}
