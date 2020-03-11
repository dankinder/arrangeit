package main

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestBasic(t *testing.T) {
	assert.Equal(t,
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
		[]*Group{
			&Group{
				Name: "Group 1",
				Items: []*Item{
					&Item{ID: "guy1", Tags: map[string]string{"gender": "m"}},
					&Item{ID: "guy2", Tags: map[string]string{"gender": "m"}},
				},
				MinSize: 1,
				MaxSize: 2,
			},
			&Group{
				Name: "Group 2",
				Items: []*Item{
					&Item{ID: "girl1", Tags: map[string]string{"gender": "f"}},
					&Item{ID: "girl2", Tags: map[string]string{"gender": "f"}},
				},
				MinSize: 1,
				MaxSize: 2,
			},
		},
	)
}

func TestWeightsWithSameness(t *testing.T) {
	assert.Equal(t,
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
		[]*Group{
			&Group{
				Name: "Group 1",
				Items: []*Item{
					&Item{ID: "guy1", Tags: map[string]string{"gender": "m", "church": "c1"}},
					&Item{ID: "guy2", Tags: map[string]string{"gender": "m", "church": "c1"}},
					&Item{ID: "guy3", Tags: map[string]string{"gender": "m", "church": "c2"}},
				},
				MinSize: 1,
				MaxSize: 3,
			},
			&Group{
				Name: "Group 2",
				Items: []*Item{
					&Item{ID: "girl1", Tags: map[string]string{"gender": "f", "church": "c1"}},
					&Item{ID: "girl2", Tags: map[string]string{"gender": "f", "church": "c2"}},
					&Item{ID: "girl3", Tags: map[string]string{"gender": "f", "church": "c2"}},
				},
				MinSize: 1,
				MaxSize: 3,
			},
		},
	)

	assert.Equal(t,
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
		[]*Group{
			&Group{
				Name: "Group 1",
				Items: []*Item{
					&Item{ID: "guy1", Tags: map[string]string{"gender": "m", "church": "c1"}},
					&Item{ID: "girl1", Tags: map[string]string{"gender": "f", "church": "c1"}},
					&Item{ID: "guy2", Tags: map[string]string{"gender": "m", "church": "c1"}},
				},
				MinSize: 1,
				MaxSize: 3,
			},
			&Group{
				Name: "Group 2",
				Items: []*Item{
					&Item{ID: "girl2", Tags: map[string]string{"gender": "f", "church": "c2"}},
					&Item{ID: "guy3", Tags: map[string]string{"gender": "m", "church": "c2"}},
					&Item{ID: "girl3", Tags: map[string]string{"gender": "f", "church": "c2"}},
				},
				MinSize: 1,
				MaxSize: 3,
			},
		},
	)
}

func TestNearness(t *testing.T) {
	assert.Equal(t,
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
		[]*Group{
			&Group{
				Name: "Group 1",
				Items: []*Item{
					&Item{ID: "guy1", Tags: map[string]string{"location": "38.831076, -77.194633"}},
					&Item{ID: "girl1", Tags: map[string]string{"location": "38.831076, -77.194633"}},
					&Item{ID: "guy2", Tags: map[string]string{"location": "38.922574, -77.235782"}},
					&Item{ID: "girl2", Tags: map[string]string{"location": "38.922574, -77.235782"}},
				},
				MinSize: 1,
				MaxSize: 4,
			},
			&Group{
				Name: "Group 2",
				Items: []*Item{
					&Item{ID: "guy3", Tags: map[string]string{"location": "38.667573, -77.255849"}},
					&Item{ID: "girl3", Tags: map[string]string{"location": "38.667573, -77.255849"}},
				},
				MinSize: 1,
				MaxSize: 4,
			},
		},
	)
}
