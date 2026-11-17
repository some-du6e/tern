// Command t3probe checks the t3 package against a live server: it pairs or
// reuses a stored session, opens the socket, reads the shell, the server
// config and one thread, and prints what it saw.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/some-du6e/tern/app/t3"
)

func main() {
	dir := flag.String("store", os.TempDir()+"/tern-probe", "store directory")
	pair := flag.String("pair", "", "pairing URL from `t3 pair`")
	flag.Parse()
	store, err := t3.OpenStoreAt(*dir)
	must(err)
	key, err := t3.LoadKey(store)
	must(err)
	var sess *t3.Session
	if *pair != "" {
		sess, err = t3.Pair(store, *pair)
		must(err)
	} else {
		envs := t3.Paired(store)
		if len(envs) == 0 {
			log.Fatal("no paired environment; pass -pair")
		}
		sess = t3.CachedSession(store, envs[0].EnvironmentID)
	}
	fmt.Println("env", sess.Label, sess.HTTPBaseURL)
	start := time.Now()
	conn, err := t3.Dial(context.Background(), key, sess)
	must(err)
	defer conn.Close()
	fmt.Println("socket open in", time.Since(start).Round(time.Millisecond))

	must(conn.Call("server.probe", map[string]any{}, nil))
	var cfg t3.ServerConfig
	must(conn.Call("server.getConfig", map[string]any{}, &cfg))
	for _, p := range cfg.Providers {
		fmt.Printf("provider %s (%s) enabled=%v status=%s models=%d\n", p.InstanceID, p.DisplayName, p.Enabled, p.Status, len(p.Models))
		for i, m := range p.Models {
			if i < 3 {
				var ds []string
				for _, d := range m.Capabilities.OptionDescriptors {
					ds = append(ds, d.ID+":"+d.Type)
				}
				fmt.Printf("   %s %q %v\n", m.Slug, m.Name, ds)
			}
		}
	}

	snap := make(chan t3.ShellItem, 1)
	var rawThread json.RawMessage
	cancel := conn.Stream("orchestration.subscribeShell", map[string]any{"requestCompletionMarker": true}, func(v json.RawMessage) {
		var it t3.ShellItem
		_ = json.Unmarshal(v, &it)
		if it.Kind == "snapshot" && it.Snapshot != nil && len(it.Snapshot.Threads) > 0 && rawThread == nil {
			var s struct {
				Snapshot struct {
					Threads []json.RawMessage `json:"threads"`
				} `json:"snapshot"`
			}
			_ = json.Unmarshal(v, &s)
			rawThread = s.Snapshot.Threads[0]
			snap <- it
		}
	}, func(err error) { log.Println("shell ended", err) })
	defer cancel()
	it := <-snap
	fmt.Printf("shell: %d projects, %d threads in %s\n", len(it.Snapshot.Projects), len(it.Snapshot.Threads), time.Since(start).Round(time.Millisecond))
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(rawThread, &keys)
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Println("thread keys:", names)

	// The most recently active top-level thread.
	var pick *t3.Thread
	for i := range it.Snapshot.Threads {
		t := &it.Snapshot.Threads[i]
		if t.Lineage != nil && t.Lineage.RelationshipToParent == "subagent" {
			continue
		}
		if pick == nil || t.UpdatedAt > pick.UpdatedAt {
			pick = t
		}
	}
	fmt.Printf("thread %q status=%s active=%s pending=%v\n", pick.Title, pick.ActivityRunStatus, pick.ActiveRunID, pick.PendingRuntimeRequest)
	got := make(chan t3.ThreadItem, 1)
	stop := conn.Stream("orchestration.subscribeThread", map[string]any{"threadId": pick.ID, "requestCompletionMarker": true, "acceptBoundedSnapshot": true, "acceptCompactTurnItems": true}, func(v json.RawMessage) {
		var ti t3.ThreadItem
		_ = json.Unmarshal(v, &ti)
		if ti.Kind == "snapshot" {
			got <- ti
		}
	}, func(error) {})
	defer stop()
	ti := <-got
	types := map[string]int{}
	for _, row := range ti.Projection.VisibleTurnItems {
		var item t3.TurnItem
		_ = json.Unmarshal(row.Item, &item)
		types[item.Type]++
	}
	fmt.Printf("thread snapshot: %d items %v, %d runs, more=%v\n", len(ti.Projection.VisibleTurnItems), types, len(ti.Projection.Runs), ti.HasMoreHistory)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
