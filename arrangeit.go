package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
)

// TODO:
//	- Implement Relationship
//	- Avoid exploring states that can't possibly meet the min-size requirements
//	- Better heuristics
//	- Specify sort preference for final output; e.g. to sort staff/drivers above students; and sort cars by bros then sis
//	- Accept another data structure for groups (e.g. the cars/vans available)

// Item defines a thing or person that has a set of tags and needs to be arranged into groups.
type Item struct {
	// ID must be unique to each item
	ID string

	// Map of tag names to tag values for this item
	Tags map[string]string

	// Maps a tag name to tag value for this item, but parsed as a point.
	// Like Tags, but only contains entries for tags that have a "Nearness" rule applied to them.
	// Used to prevent having to re-parse these entries over and over.
	nearnessTags map[string]point
}

// RuleType definitions control the behavior of a rule and can be found below.
type RuleType string

const (
	// Try to keep items together that share the same value for this tag.
	RuleTypeSameness RuleType = "Sameness"

	// Interpret the tag value as the ID of another item, and try to keep these items together.
	RuleTypeRelationship RuleType = "Relationship"

	// Try to interpret the given tag value as a geolocation and put nearby items together.
	RuleTypeNearness RuleType = "Nearness"
)

// Rule is one instance of an input rule. There could potentially be multiple rules on the same tag and/or of the same
// type.
type Rule struct {
	// Tag name to which this rule applies
	TagName string

	// What type of rule this is (see RuleType)
	Type RuleType

	// How important this rule is relative to the other rules
	Weight int
}

// Group is passed to GetArrangement to indicate what groups there are and how full they can be.
// Items will be populated by GetArrangement.
type Group struct {
	Name    string
	MinSize int
	MaxSize int
	Items   []*Item
}

// digest produces a unique hash digest of the group, intended such that groups that are "equivalent" (e.g. regardless
// of ordering, treating people with identical attributes as the same, etc.) have the same digest.
func (g *Group) digest() uint64 {
	itemsSorted := append([]*Item(nil), g.Items...)
	sort.Slice(itemsSorted, func(i, j int) bool { return itemsSorted[i].ID < itemsSorted[j].ID })
	h := fnv.New64()
	for _, item := range itemsSorted {
		h.Write([]byte(item.ID))
	}
	return h.Sum64()
}

// Copy creates a copy of a Group so it can be modified for a new State.
// Note that Items are not deep copied as we don't modify these.
func (g *Group) Copy() *Group {
	newGroup := &Group{
		Name:    g.Name,
		MinSize: g.MinSize,
		MaxSize: g.MaxSize,
		Items:   make([]*Item, 0, len(g.Items)),
	}
	newGroup.Items = append(newGroup.Items, g.Items...)
	return newGroup
}

// MustGetArrangement calls GetArrangement but panics on failures. Good for testing.
func MustGetArrangement(items []*Item, rules []*Rule, groups []*Group) []*Group {
	result, err := GetArrangement(context.Background(), items, rules, groups)
	if err != nil {
		panic(fmt.Sprintf("GetArrangement failed: %v", err))
	}
	return result
}

// GetArrangement is the primary workhorse of the algorithm. Given a set of items, rules, and groups to fill, it returns
// copies of the Groups with Items filled in matching the rules.
func GetArrangement(ctx context.Context, items []*Item, rules []*Rule, groups []*Group) ([]*Group, error) {
	r := runner{
		ctx:                      ctx,
		items:                    items,
		rules:                    rules,
		groups:                   groups,
		maxDistributionByTagName: map[string]float64{},
		statesTried:              map[uint64]struct{}{},
	}
	return r.run()
}

// State represents a particular arrangement of items in the groups, which may be intermediate/non-terminal. I.e. not
// everyone has been placed in a group yet.
type State struct {
	// Groups which include items that may have been put in them in this state
	Groups []*Group

	// Items that haven't been put into groups yet for this state
	ItemsNotInGroups []*Item

	// Score is higher the better this State follows the provided rules.
	// If this state is non-terminal (i.e. not all items are in groups yet), then Score is a heuristically guessed
	// maximum score this state may end up producing after further iteration.
	Score float64
}

// digest produces a unique hash digest of the state, intended such that states that are "equivalent" (e.g. regardless
// of ordering, treating people with identical attributes as the same, etc.) have the same digest.
func (s *State) digest() uint64 {
	// First sort the digests, since we want the same digest regardless of the order of the groups
	var digests []uint64
	for _, group := range s.Groups {
		digests = append(digests, group.digest())
	}
	sort.Slice(digests, func(i, j int) bool { return digests[i] < digests[j] })

	h := fnv.New64()
	buf := make([]byte, binary.MaxVarintLen64)
	for _, d := range digests {
		binary.PutUvarint(buf, d)
		h.Write(buf)
	}
	return h.Sum64()
}

// Copy produces a new State that can be modified without messing with the old State.
// NOTE: Items themselves are not deep copied since we don't modify those.
func (s *State) Copy() *State {
	// Note: we don't have to copy the items themselves in these objects because we don't modify them
	newState := &State{}
	newState.ItemsNotInGroups = append(newState.ItemsNotInGroups, s.ItemsNotInGroups...)
	newState.Groups = make([]*Group, 0, len(s.Groups))
	for _, group := range s.Groups {
		newState.Groups = append(newState.Groups, group.Copy())
	}
	return newState
}

// IsTerminal returns true if this state is a complete arrangement, every item has been put in a group.
func (s *State) IsTerminal() bool {
	return len(s.ItemsNotInGroups) == 0
}

//
// The main algorithm runner
//

type runner struct {
	// Stuff to be initialized with. Note that these slices should not be modified during the algorithm.
	//
	ctx    context.Context
	items  []*Item
	rules  []*Rule
	groups []*Group

	// Stuff created along the way:
	//
	// Best terminal state we've found along the way (cannot be a non-terminal state)
	bestState   *State
	statesToTry []*State

	// Used for caching the maximum distribution in location/nearness calculations
	maxDistributionByTagName map[string]float64

	// Maps a state digest to the score we got for that state
	statesTried map[uint64]struct{}

	// For state generation, the current permutation of items we're trying
	currentPermutation []int
}

func (r *runner) run() ([]*Group, error) {
	if err := r.validateInput(); err != nil {
		return nil, err
	}

	r.populateNearnessTagPoints()
	defer r.clearNearnessTagPoints()

	next := r.getRandomState()
	r.bestState = next

	for {
		if r.quitting() {
			break
		}

		digest := next.digest()
		if _, ok := r.statesTried[digest]; ok {
			next = r.getRandomState()
			if next == nil {
				break
			}
			continue
		}
		r.statesTried[digest] = struct{}{}

		bestOption := r.getBestNextStateFrom(next)
		if bestOption.Score > next.Score {
			// Keep exploring starting from this new best state
			next = bestOption
			continue
		}

		if next.Score > r.bestState.Score {
			log.Println("Found better state")
			r.bestState = next
		}

		// At this point we've explored `next` up to a local maximum score, now let's restart from a random spot and see
		// if we find anything better
		next = r.getRandomState()
		if next == nil {
			break
		}
	}

	return r.bestState.Groups, nil
}

func (r *runner) quitting() bool {
	// Check if we've timed out. Will cause us to return the best state we have so far.
	select {
	case <-r.ctx.Done():
		return true
	default:
		return false
	}
}

// getRandomState keeps returning different permutations of possible states.
// It will never repeat the same state twice, and when it has exhausted all possible permutations it will return nil.
func (r *runner) getRandomState() *State {
	if r.currentPermutation == nil {
		// On our first pass, use an empty permutation, which just means return the items in existing order
		r.currentPermutation = make([]int, len(r.items))
	} else {
		// Increment to the next permutation
		// For now this is a fisher-yates algorithm, as provided in https://stackoverflow.com/a/30230552
		for i := len(r.currentPermutation) - 1; i >= 0; i-- {
			if i == 0 || r.currentPermutation[i] < len(r.currentPermutation)-i-1 {
				r.currentPermutation[i]++
				break
			}
			r.currentPermutation[i] = 0
		}

		if r.currentPermutation[0] >= len(r.currentPermutation) {
			// This indicates we've gone through every permutation
			return nil
		}
	}

	nextPerm := append([]*Item{}, r.items...)
	for i, v := range r.currentPermutation {
		nextPerm[i], nextPerm[i+v] = nextPerm[i+v], nextPerm[i]
	}

	// Given a permutation of items now, scatter them evenly across the groups
	s := &State{
		Groups: make([]*Group, 0, len(r.groups)),
	}
	for _, group := range r.groups {
		s.Groups = append(s.Groups, &Group{
			Name:    group.Name,
			MinSize: group.MinSize,
			MaxSize: group.MaxSize,
			Items:   make([]*Item, 0, len(r.items)/len(r.groups)),
		})
	}

	// First, ensure every group has at least MinSize number of items
	i := 0
	for _, group := range s.Groups {
		for i < len(nextPerm) && len(group.Items) < group.MinSize {
			group.Items = append(group.Items, nextPerm[i])
			i++
		}
	}

	// Now add people to groups round-robin
	for i < len(nextPerm) {
		for _, group := range s.Groups {
			// NOTE: this could loop forever if there isn't enough room for everyone; but we have validation to ensure
			// that can't happen
			if len(group.Items) == group.MaxSize {
				// This group is maxed, we can't try putting another in it
				continue
			}

			group.Items = append(group.Items, nextPerm[i])
			i++
			if i >= len(nextPerm) {
				break
			}
		}
	}
	s.Score = r.CalculateScore(s)

	return s
}

func (r *runner) validateInput() error {
	var numSlots int
	for _, group := range r.groups {
		numSlots += group.MaxSize
	}
	if numSlots < len(r.items) {
		return fmt.Errorf("bad configuration: there are %d items to arrange but only %d possible slots", len(r.items), numSlots)
	}
	return nil
}

func (r *runner) getBestNextStateFrom(sourceState *State) *State {
	// NOTE: we'd quit faster by checking `quitting()` in the loop here, but would also slow us down

	// Here we loop through all items, trying all the possible ways we can move them around.
	// For groups that aren't maxed out, we just try moving the item into the group.
	// For groups that are maxed we need to try swapping our item with one already in the group.

	bestOption := sourceState
	// Copy the sourceState, so we aren't messing with it as we move stuff around in `s`
	s := sourceState.Copy()

	for gIndex1 := range s.Groups {
		for i := 0; i < len(s.Groups[gIndex1].Items); i++ {

			for gIndex2 := range s.Groups {

				// We need to redefine tehse every loop iteration because below, we mess with the groups
				g1, g2 := s.Groups[gIndex1], s.Groups[gIndex2]

				if g1 == g2 {
					continue
				}

				if len(g2.Items) < g2.MaxSize {
					// The group isn't full yet, try moving our current item into it
					origG1Items := g1.Items
					origG2Items := g2.Items

					// NOTE: we don't need to make a copy of this since we aren't modifying any of the cells it points to
					g2.Items = append(g2.Items, g1.Items[i])

					// Make a copy of g1.Items and delete the item by overwriting it with the last item
					g1.Items = append([]*Item(nil), g1.Items...)
					g1.Items[i] = g1.Items[len(g1.Items)-1]
					g1.Items = g1.Items[:len(g1.Items)-1]

					s.Score = r.CalculateScore(s)
					if s.Score > bestOption.Score {
						bestOption = s
						s = sourceState.Copy()
					} else {
						// We're going to reuse s, so restore it to how it was
						g1.Items = origG1Items
						g2.Items = origG2Items
					}

				} else {
					// This group is full, so we need to try a swap with each person in it
					// TODO: currently we waste effort since if 2 groups are full, we'll try swapping every person in
					// each with the other, twice. We should change this to try swapping people from earlier groups with
					// later groups, but not vice versa.
					for i2 := 0; i2 < len(g2.Items); i2++ {
						g1.Items[i], g2.Items[i2] = g2.Items[i2], g1.Items[i]

						s.Score = r.CalculateScore(s)
						if s.Score > bestOption.Score {
							bestOption = s
							s = sourceState.Copy()
						} else {
							// We're going to reuse s, so restore it to how it was
							g1.Items[i], g2.Items[i2] = g2.Items[i2], g1.Items[i]
						}
					}
				}
			}
		}
	}
	return bestOption
}

// insertStateToTry adds in the new state while maintaining that states is sorted from highest to lowest score
func (r *runner) insertStateToTry(states []*State, toInsert *State) []*State {
	i := sort.Search(len(states), func(i int) bool {
		return states[i].Score < toInsert.Score
	})
	if i < len(states) {
		// Insert the new state in the position it should be
		states = append(states[:i], append([]*State{toInsert}, states[i:]...)...)
	} else {
		states = append(states, toInsert)
	}
	return states
}

func (r *runner) CalculateScore(s *State) float64 {
	// If a state is not terminal then calculate a heuristic rather than a real score
	if !s.IsTerminal() {
		return r.CalculateMaxPotentialScore(s)
	}

	// For terminal states, return the lowest possible score if it doesn't meet minimum group size constraints
	for i := 0; i < len(s.Groups); i++ {
		if len(s.Groups[i].Items) > 0 && len(s.Groups[i].Items) < s.Groups[i].MinSize {
			return -math.MaxFloat64
		}
	}

	return r.CalculateCurrentScore(s)
}

func (r *runner) CalculateCurrentScore(s *State) float64 {
	var score float64
	for _, rule := range r.rules {
		if rule.Weight == 0 {
			continue
		}

		switch rule.Type {
		case RuleTypeSameness:
			for _, group := range s.Groups {
				tagOccurrencesInGroup := map[string]int{}
				for _, item := range group.Items {
					val := item.Tags[rule.TagName]
					if val == "" {
						continue
					}
					tagOccurrencesInGroup[val]++
				}
				for _, count := range tagOccurrencesInGroup {
					// Increase the score by count squared in order to prefer that many people with the same tag be
					// together.
					score += float64(rule.Weight) * math.Pow(float64(count), 2)
				}
			}
		case RuleTypeRelationship:
			panic("RuleTypeRelationship not yet implemented")
		case RuleTypeNearness:
			for _, group := range s.Groups {
				// We score "nearness" by getting a distribution ratio for the points in the group, relative to the
				// distribution of all points in all items. I.e. if the current group's items are within a very small
				// distance of each other, much smaller than the general distribution of points, then the
				// distributionRatio will be close to 0. If they are far apart it'll be near 1. (Smaller is better)
				distribution, numPoints := getGroupDistribution(group, rule.TagName)
				distributionRatio := distribution / r.maxDistributionForTag(rule.TagName)

				// This scoring rewards many points being together that still have a low distribution ratio.
				score += float64(rule.Weight) * float64(numPoints) * (1 - distributionRatio)
			}
		}
	}
	return score
}

func (r *runner) CalculateMaxPotentialScore(s *State) float64 {
	maxScore := r.CalculateCurrentScore(s)

	for _, rule := range r.rules {
		if rule.Weight == 0 {
			continue
		}

		switch rule.Type {
		case RuleTypeSameness:
			// If the rule weight is negative, the best we could theoretically do is keep everyone with this tag
			// separate, which would result in a score of 0
			// TODO: we could do better but this will be fine for now
			if rule.Weight < 0 {
				continue
			}

			// TODO: make this heuristic much smarter, some ideas below
			maxScore += float64(rule.Weight * len(s.ItemsNotInGroups))

			//// First, figure out what tag values are left in the unassigned items, and how many
			//tagOccurrencesTotal := map[string]int{}
			//for _, item := range s.ItemsNotInGroups {
			//	val := item.Tags[rule.TagName]
			//	if tagValue == "" {
			//		continue
			//	}
			//	tagOccurrencesTotal[val]++
			//}

			//for val := range tagOccurrencesTotal {
			//}

			//// Otherwise, the theoretical maximum score for a sameness would be if we got everyone with the same tag
			//// values to be in the same group together. So count that up and that's our max
			//for _, group := range s.Groups {
			//	for _, item := range group.Items {
			//		val := item.Tags[rule.TagName]
			//		if tagValue == "" {
			//			continue
			//		}
			//		tagOccurrencesTotal[val]++
			//	}
			//}

			//for _, count := range tagOccurrencesInGroup {
			//	// We want to subtract 1 here because we only want to add to the score if at least 2 people actually
			//	// share the same tag value.
			//	maxScore += float64(rule.Weight * (count - 1))
			//}

		case RuleTypeRelationship:
			panic("RuleTypeRelationship not yet implemented")

		case RuleTypeNearness:
			// If the rule weight is negative, the best we could theoretically do is keep the score at 0
			if rule.Weight < 0 {
				continue
			}

			var itemsWithPoints int
			for _, item := range s.ItemsNotInGroups {
				if _, ok := item.nearnessTags[rule.TagName]; ok {
					itemsWithPoints++
				}
			}

			// The absolute maximum score here is `rule.Weight * itemsWithPoints`, assuming that each item still to be
			// placed is placed with a group that gets maximum score for nearness.
			// But we can guarantee the max score is lower than that for groups that already have a non-0 distribution.
			// Since the distribution can only go up, any new point added to such groups won't be worth a full
			// `rule.Weight`, it'll be worth less, depending on how distributed that group is.
			// So we go through each group gathering the distribution and the number of slots left, sort them so the
			// lowest distribution ones are first, then "fill" them in order.
			type groupToFill struct {
				distribution float64
				slotsLeft    int
			}
			groupsToFill := make([]groupToFill, 0, len(s.Groups))
			for _, group := range s.Groups {
				distribution, _ := getGroupDistribution(group, rule.TagName)
				slotsLeft := group.MaxSize - len(group.Items)
				if slotsLeft > 0 {
					groupsToFill = append(groupsToFill, groupToFill{distribution, slotsLeft})
				}
			}
			sort.Slice(groupsToFill, func(i, j int) bool { return groupsToFill[i].distribution < groupsToFill[j].distribution })

			maxDist := r.maxDistributionForTag(rule.TagName)
			for i := 0; itemsWithPoints > 0 && i < len(groupsToFill); i++ {
				groupToFill := groupsToFill[i]

				var numToFill int
				if itemsWithPoints >= groupToFill.slotsLeft {
					numToFill = groupToFill.slotsLeft
				} else {
					numToFill = itemsWithPoints
				}
				itemsWithPoints -= numToFill

				// For an explanation of this calculation see CalculateCurrentScore
				distributionRatio := groupToFill.distribution / maxDist
				maxScore += float64(rule.Weight) * float64(numToFill) * (1 - distributionRatio)
			}
		}
	}
	return maxScore
}

// Functions for calculating geolocation/distribution
//

type point struct {
	x float64
	y float64
}

func (r *runner) parsePoint(str string) (point, error) {
	parts := strings.Split(str, ",")
	if len(parts) != 2 {
		return point{}, fmt.Errorf("failed to interpret %q as x/y coordinate", str)
	}
	var err error
	var p point
	p.x, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return point{}, fmt.Errorf("failed to parse x coordinate of %q: %v", str, err)
	}
	p.y, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return point{}, fmt.Errorf("failed to parse y coordinate of %q: %v", str, err)
	}
	return p, nil
}

func (r *runner) populateNearnessTagPoints() {
	for _, rule := range r.rules {
		if rule.Weight == 0 || rule.Type != RuleTypeNearness {
			continue
		}

		for _, item := range r.items {
			val := item.Tags[rule.TagName]
			if val == "" {
				continue
			}

			p, err := r.parsePoint(val)
			if err != nil {
				// When we calculate the total distribution we'll log these, so don't do it here, just skip
				continue
			}

			if item.nearnessTags == nil {
				item.nearnessTags = map[string]point{}
			}
			item.nearnessTags[rule.TagName] = p
		}
	}
}

// clearNearnessTagPoints blows away item.nearnessTags so that in the tests we can use assert.Equals on Item objects
// easily.
func (r *runner) clearNearnessTagPoints() {
	for _, item := range r.items {
		item.nearnessTags = nil
	}
}

// getGroupDistribution returns the distribution of the points in the provided group (see getDistribution) along with
// the number of points there are. It copies some of getDistribution for performance reasons.
func getGroupDistribution(group *Group, tagName string) (float64, int) {
	var maxX, maxY, minX, minY float64
	var numPoints int

	for _, item := range group.Items {
		if p, ok := item.nearnessTags[tagName]; ok {
			if numPoints > 0 {
				if p.x > maxX {
					maxX = p.x
				}
				if p.x < minX {
					minX = p.x
				}
				if p.y > maxY {
					maxY = p.y
				}
				if p.y < minY {
					minY = p.y
				}
			} else {
				maxX = p.x
				minX = p.x
				maxY = p.y
				minY = p.y
			}
			numPoints++
		}
	}

	if numPoints > 0 {
		return (maxX - minX) + (maxY - minY), numPoints
	} else {
		return 0, numPoints
	}
}

// getDistribution calculates how distributed the provided points are.
// For the moment it just figures out the smallest square (min X,Y and max X,Y) that captures all the points and returns
// the width + height of the box.
func getDistribution(points []point) float64 {
	if len(points) == 0 {
		return 0
	}
	maxX, maxY := points[0].x, points[0].y
	minX, minY := maxX, maxY
	for _, p := range points[1:] {
		if p.x > maxX {
			maxX = p.x
		}
		if p.x < minX {
			minX = p.x
		}
		if p.y > maxY {
			maxY = p.y
		}
		if p.y < minY {
			minY = p.y
		}
	}
	return (maxX - minX) + (maxY - minY)
}

func (r *runner) maxDistributionForTag(tagName string) float64 {
	if cachedVal, ok := r.maxDistributionByTagName[tagName]; ok {
		return cachedVal
	}

	var points []point
	for _, item := range r.items {
		val := item.Tags[tagName]
		if val == "" {
			continue
		}
		p, err := r.parsePoint(val)
		if err != nil {
			log.Printf("Failed to parse point: %v", err)
			continue
		}
		points = append(points, p)
	}
	dist := getDistribution(points)
	r.maxDistributionByTagName[tagName] = dist
	return dist
}

// Extra stuff
//

func factorial(n int) int {
	if n <= 0 {
		return 1
	}

	final := 1
	for i := 1; i <= n; i++ {
		final *= i
	}
	return final
}
