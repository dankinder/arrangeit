package main

import (
	"sort"
	"testing"

	"github.com/bmizerany/assert"
)

// assertArrangementsEqual compares two arrangements ignoring data fields we don't care about and focusing on the items
// being grouped properly. Ignoring sort orders and such.
func assertArrangementsEqual(t *testing.T, exp, got []*Group) {
	// Sort all the items in both groups, then the groups themselves, in a stable way for comparison
	for _, groups := range [][]*Group{exp, got} {
		for _, group := range groups {
			sort.Slice(group.Items, func(i, j int) bool { return group.Items[i].ID < group.Items[j].ID })
		}
		sort.Slice(groups, func(i, j int) bool {
			if len(groups[i].Items) != len(groups[j].Items) {
				return len(groups[i].Items) < len(groups[j].Items)
			}
			if len(groups[i].Items) == 0 && len(groups[j].Items) == 0 {
				return false
			}
			return groups[i].Items[0].ID < groups[j].Items[0].ID
		})
	}

	for i := range exp {
		var expItemIDs []string
		for _, item := range exp[i].Items {
			expItemIDs = append(expItemIDs, item.ID)
		}
		var gotItemIDs []string
		for _, item := range got[i].Items {
			gotItemIDs = append(gotItemIDs, item.ID)
		}
		assert.Equal(t, expItemIDs, gotItemIDs)
	}
}

func TestBasic(t *testing.T) {
	assertArrangementsEqual(t,
		[]*Group{
			&Group{Items: []*Item{&Item{ID: "girl2"}, &Item{ID: "girl1"}}},
			&Group{Items: []*Item{&Item{ID: "guy2"}, &Item{ID: "guy1"}}},
		},
		MustGetArrangement(
			[]*Item{
				&Item{ID: "guy1", Tags: map[string]string{"gender": "m"}},
				&Item{ID: "girl1", Tags: map[string]string{"gender": "f"}},
				&Item{ID: "guy2", Tags: map[string]string{"gender": "m"}},
				&Item{ID: "girl2", Tags: map[string]string{"gender": "f"}},
			},
			[]*Rule{
				&Rule{TagName: "gender", Type: RuleTypeSameness, Weight: 1},
			},
			[]*Group{
				&Group{Name: "Group 1", MinSize: 1, MaxSize: 2},
				&Group{Name: "Group 2", MinSize: 1, MaxSize: 2},
			}),
	)
}

func TestWeightsWithSamenessGenderThenChurch(t *testing.T) {
	assertArrangementsEqual(t,
		[]*Group{
			&Group{
				Items: []*Item{&Item{ID: "girl3"}, &Item{ID: "girl1"}, &Item{ID: "girl2"}},
			},
			&Group{
				Items: []*Item{&Item{ID: "guy3"}, &Item{ID: "guy2"}, &Item{ID: "guy1"}},
			},
		},
		MustGetArrangement(
			[]*Item{
				&Item{ID: "guy1", Tags: map[string]string{"gender": "m", "church": "c1"}},
				&Item{ID: "girl1", Tags: map[string]string{"gender": "f", "church": "c1"}},
				&Item{ID: "guy2", Tags: map[string]string{"gender": "m", "church": "c1"}},
				&Item{ID: "girl2", Tags: map[string]string{"gender": "f", "church": "c2"}},
				&Item{ID: "guy3", Tags: map[string]string{"gender": "m", "church": "c2"}},
				&Item{ID: "girl3", Tags: map[string]string{"gender": "f", "church": "c2"}},
			},
			[]*Rule{
				&Rule{TagName: "gender", Type: RuleTypeSameness, Weight: 2},
				&Rule{TagName: "church", Type: RuleTypeSameness, Weight: 1},
			},
			[]*Group{
				&Group{Name: "Group 1", MinSize: 1, MaxSize: 3},
				&Group{Name: "Group 2", MinSize: 1, MaxSize: 3},
			}),
	)
}

func TestWeightsWithSamenessChurchThenGender(t *testing.T) {
	assertArrangementsEqual(t,
		[]*Group{
			&Group{
				Items: []*Item{&Item{ID: "girl3"}, &Item{ID: "guy3"}, &Item{ID: "girl2"}},
			},
			&Group{
				Items: []*Item{&Item{ID: "guy2"}, &Item{ID: "girl1"}, &Item{ID: "guy1"}},
			},
		},
		MustGetArrangement(
			[]*Item{
				&Item{ID: "guy1", Tags: map[string]string{"gender": "m", "church": "c1"}},
				&Item{ID: "girl1", Tags: map[string]string{"gender": "f", "church": "c1"}},
				&Item{ID: "guy2", Tags: map[string]string{"gender": "m", "church": "c1"}},
				&Item{ID: "girl2", Tags: map[string]string{"gender": "f", "church": "c2"}},
				&Item{ID: "guy3", Tags: map[string]string{"gender": "m", "church": "c2"}},
				&Item{ID: "girl3", Tags: map[string]string{"gender": "f", "church": "c2"}},
			},
			[]*Rule{
				&Rule{TagName: "gender", Type: RuleTypeSameness, Weight: 1},
				&Rule{TagName: "church", Type: RuleTypeSameness, Weight: 2},
			},
			[]*Group{
				&Group{Name: "Group 1", MinSize: 1, MaxSize: 3},
				&Group{Name: "Group 2", MinSize: 1, MaxSize: 3},
			}),
	)
}

func TestNearness(t *testing.T) {
	assertArrangementsEqual(t,
		[]*Group{
			&Group{
				Items: []*Item{&Item{ID: "girl3"}, &Item{ID: "guy3"}},
			},
			&Group{
				Items: []*Item{&Item{ID: "girl2"}, &Item{ID: "girl1"}, &Item{ID: "guy2"}, &Item{ID: "guy1"}},
			},
		},
		MustGetArrangement(
			[]*Item{
				// 38.831076, -77.194633 is Annandale
				&Item{ID: "guy1", Tags: map[string]string{"location": "38.831076, -77.194633"}},
				&Item{ID: "girl1", Tags: map[string]string{"location": "38.831076, -77.194633"}},
				// 38.922574, -77.235782 is Tysons
				&Item{ID: "guy2", Tags: map[string]string{"location": "38.922574, -77.235782"}},
				&Item{ID: "girl2", Tags: map[string]string{"location": "38.922574, -77.235782"}},
				// 38.667573, -77.255849 is Woodbridge
				&Item{ID: "guy3", Tags: map[string]string{"location": "38.667573, -77.255849"}},
				&Item{ID: "girl3", Tags: map[string]string{"location": "38.667573, -77.255849"}},
			},
			[]*Rule{
				&Rule{TagName: "location", Type: RuleTypeNearness, Weight: 1},
			},
			[]*Group{
				&Group{Name: "Group 1", MinSize: 1, MaxSize: 4},
				&Group{Name: "Group 2", MinSize: 1, MaxSize: 4},
			}),
	)
}

func TestRespectMinSize(t *testing.T) {
	assertArrangementsEqual(t,
		[]*Group{
			&Group{
				Name:  "Group 1",
				Items: []*Item{&Item{ID: "guy1"}, &Item{ID: "girl1"}, &Item{ID: "guy2"}, &Item{ID: "girl2"}},
			},
			&Group{
				Items: []*Item{},
			},
		},
		MustGetArrangement(
			[]*Item{
				&Item{ID: "girl2", Tags: map[string]string{"gender": "f"}},
				&Item{ID: "guy2", Tags: map[string]string{"gender": "m"}},
				&Item{ID: "girl1", Tags: map[string]string{"gender": "f"}},
				&Item{ID: "guy1", Tags: map[string]string{"gender": "m"}},
			},
			[]*Rule{
				&Rule{TagName: "gender", Type: RuleTypeSameness, Weight: 1},
			},
			[]*Group{
				&Group{Name: "Group 1", MinSize: 3, MaxSize: 4},
				&Group{Name: "Group 2", MinSize: 3, MaxSize: 4},
			}),
	)
}
