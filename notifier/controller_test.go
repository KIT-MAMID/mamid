package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDiffProblemsNew(t *testing.T) {
	lastProblems = make(map[uint]Problem)
	newProblems := []Problem{Problem{Id: 1}, Problem{Id: 2}, Problem{Id: 3}, Problem{Id: 4}}
	assert.Equal(t, newProblems, diffProblems(newProblems))
}

func TestDiffProblemsNewProblem(t *testing.T) {
	lastProblems = make(map[uint]Problem)
	newProblems := []Problem{Problem{Id: 1}, Problem{Id: 2}, Problem{Id: 3}, Problem{Id: 4}}
	assert.Equal(t, newProblems, diffProblems(newProblems))
	newProblems = []Problem{Problem{Id: 1}, Problem{Id: 2}, Problem{Id: 3}, Problem{Id: 4}, Problem{Id: 5}}
	assert.Equal(t, []Problem{newProblems[4]}, diffProblems(newProblems))
}

func TestDiffProblemsNewOtherProblems(t *testing.T) {
	lastProblems = make(map[uint]Problem)
	newProblems := []Problem{Problem{Id: 1}, Problem{Id: 2}, Problem{Id: 3}, Problem{Id: 4}}
	assert.Equal(t, newProblems, diffProblems(newProblems))
	newProblems = []Problem{Problem{Id: 6}, Problem{Id: 7}, Problem{Id: 8}, Problem{Id: 9}, Problem{Id: 10}}
	assert.Equal(t, newProblems, diffProblems(newProblems))
}

func TestDiffProblemsLessProblems(t *testing.T) {
	lastProblems = make(map[uint]Problem)
	newProblems := []Problem{Problem{Id: 1}, Problem{Id: 2}, Problem{Id: 3}, Problem{Id: 4}}
	assert.Equal(t, newProblems, diffProblems(newProblems))
	newProblems = []Problem{Problem{Id: 1}}
	assert.Equal(t, []Problem{}, diffProblems(newProblems))
}
