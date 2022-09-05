/*

Package listen is a self-contained Go program which uses the LISTEN / NOTIFY
mechanism to avoid polling the database while waiting for more work to arrive.

    //
    // You can see the program in action by defining a function similar to
    // the following:
    //
    // CREATE OR REPLACE FUNCTION public.get_work()
    //   RETURNS bigint
    //   LANGUAGE sql
    //   AS $$
    //     SELECT CASE WHEN random() >= 0.2 THEN int8 '1' END
    //   $$
    // ;
*/
package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

func waitForNotification(l *pq.Listener) {
	select {
	case n, _ := <-l.Notify:
		fmt.Println(n.Extra)
	}
}

func main() {
	var conninfo string = "postgres://ionscale:ionscale@localhost/ionscale?sslmode=disable"

	_, err := sql.Open("postgres", conninfo)
	if err != nil {
		panic(err)
	}

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	minReconn := 10 * time.Second
	maxReconn := time.Minute
	listener := pq.NewListener(conninfo, minReconn, maxReconn, reportProblem)
	err = listener.Listen("ionscale_events")
	if err != nil {
		panic(err)
	}

	fmt.Println("entering main loop")
	for {
		// process all available work before waiting for notifications
		waitForNotification(listener)
	}
}
