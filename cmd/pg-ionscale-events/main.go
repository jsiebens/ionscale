package main

import (
	"database/sql"
	"fmt"
	"github.com/muesli/coral"
	"os"
	"time"

	"github.com/lib/pq"
)

func main() {
	cmd := rootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCommand() *coral.Command {
	command := &coral.Command{
		Use: "pg-ionscale-events",
	}

	var url string
	command.Flags().StringVar(&url, "url", "", "")
	_ = command.MarkFlagRequired("url")

	command.RunE = func(cmd *coral.Command, args []string) error {
		_, err := sql.Open("postgres", url)
		if err != nil {
			return err
		}

		reportProblem := func(ev pq.ListenerEventType, err error) {
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		minReconn := 10 * time.Second
		maxReconn := time.Minute
		listener := pq.NewListener(url, minReconn, maxReconn, reportProblem)
		err = listener.Listen("ionscale_events")
		if err != nil {
			return err
		}

		fmt.Println("listening for events ...")
		fmt.Println("")
		for {
			select {
			case n, _ := <-listener.Notify:
				fmt.Println(n.Extra)
			}
		}
	}

	return command
}
