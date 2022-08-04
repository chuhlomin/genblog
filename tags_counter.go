package main

import (
	"log"
	"sort"
)

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

type TagsCounterList map[string]int

func (tcl TagsCounterList) Add(tags []string) {
	for _, tag := range tags {
		tcl[tag]++
	}
}

func printTagsStags(tagsCounter TagsCounterList) {
	log.Println("Tags counts:")
	p := make(PairList, len(tagsCounter))
	i := 0
	for k, v := range tagsCounter {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(p))

	for _, k := range p {
		log.Printf("  %v: %v", k.Key, k.Value)
	}
}
