package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dankinder/handle"
)

var itemsFile string
var rulesFile string
var groupsFile string

var minGroupSize int
var maxGroupSize int
var maxNumGroups int

var timeoutSeconds int

func init() {
	flag.StringVar(&itemsFile, "items", "", "path to the items to arrange")
	flag.StringVar(&rulesFile, "rules", "", "path to the rules file")
	flag.StringVar(&groupsFile, "groups", "", "path to the list of groups")
	flag.IntVar(&minGroupSize, "min-size", 0, "path to the rules file")
	flag.IntVar(&maxGroupSize, "max-size", 0, "maximum size of a group")
	flag.IntVar(&maxNumGroups, "max-groups", 0, "maximum number of groups")
	flag.IntVar(&timeoutSeconds, "timeout-secs", 0, "after this many seconds, return the best arrangement found so far")
}

// TODO better help text

func main() {
	flag.Parse()
	if itemsFile == "" || rulesFile == "" {
		fmt.Println("-items and -rules are required")
		os.Exit(1)
	}

	if groupsFile == "" && (maxGroupSize == 0 && maxNumGroups == 0) {
		fmt.Println("either -groups or -max-size and -max-groups are required")
		os.Exit(1)
	}

	items := readItemsFromCSV(itemsFile)
	rules := readRulesFromCSV(rulesFile)

	var groups []*Group
	if groupsFile != "" {
		groups = readGroupsFromCSV(groupsFile)
	} else {
		for i := 0; i < maxNumGroups; i++ {
			groups = append(groups, &Group{Name: fmt.Sprintf("Group %d", i+1), MaxSize: maxGroupSize, MinSize: minGroupSize})
		}
	}

	pprofPath := os.Getenv("CPU_PROFILE_PATH")
	if pprofPath != "" {
		f, err := os.Create(pprofPath)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	ctx := context.Background()
	if timeoutSeconds != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeoutSeconds))
		defer cancel()
	}

	arrangement, err := GetArrangement(ctx, items, rules, groups)
	if err != nil {
		fmt.Printf("error computing arrangement: %v\n", err)
		os.Exit(1)
	}

	for _, group := range arrangement {
		fmt.Println("---")
		fmt.Println(group.Name)
		for _, item := range group.Items {
			var tags []string
			for tagName, tagValue := range item.Tags {
				tags = append(tags, fmt.Sprintf("%s=%s", tagName, tagValue))
			}
			sort.Strings(tags)
			fmt.Printf("    - %s (%s)\n", item.ID, strings.Join(tags, " "))

		}
	}
}

func getRecords(csvPath string) [][]string {
	f, err := os.Open(csvPath)
	handle.Err(err)
	records, err := csv.NewReader(f).ReadAll()
	handle.Err(err)

	if len(records) < 1 {
		fmt.Println("at least a header row is required in " + csvPath)
		os.Exit(1)
	}
	return records
}

func readItemsFromCSV(csvPath string) []*Item {
	records := getRecords(csvPath)

	// The first record is the header row; the first column is assumed to be the ID, so the rest are tag names
	columnNames := records[0][1:]
	records = records[1:]

	var items []*Item
	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		item := &Item{ID: record[0], Tags: map[string]string{}}
		for i, columnValue := range record[1:] {
			item.Tags[columnNames[i]] = columnValue
		}
		items = append(items, item)
	}
	return items
}

func readRulesFromCSV(csvPath string) []*Rule {
	records := getRecords(csvPath)
	columnNames := records[0]
	records = records[1:]

	var rules []*Rule
	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		rule := &Rule{}
		for i, columnValue := range record {
			switch columnNames[i] {
			case "TagName":
				rule.TagName = columnValue
			case "RuleType":
				rule.Type = RuleType(columnValue)
			case "Weight":
				var err error
				rule.Weight, err = strconv.Atoi(columnValue)
				handle.Err(err)
			}
		}
		rules = append(rules, rule)
	}
	return rules
}

func readGroupsFromCSV(csvPath string) []*Group {
	records := getRecords(csvPath)
	columnNames := records[0]
	records = records[1:]

	var groups []*Group
	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		var err error
		group := &Group{}
		for i, columnValue := range record {
			switch columnNames[i] {
			case "GroupName":
				group.Name = columnValue
			case "MinSize":
				group.MinSize, err = strconv.Atoi(columnValue)
				handle.Err(err)
			case "MaxSize":
				group.MaxSize, err = strconv.Atoi(columnValue)
				handle.Err(err)
			}
		}
		groups = append(groups, group)
	}
	return groups
}
