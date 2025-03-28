package state

type QueryHasCyclesErr struct {
	Cycles []Cycle
}

func (e QueryHasCyclesErr) Error() string {
	return "uh oh there are cycles but I'm too lazy to print them out"
}

type Cycle struct {
	Members []Package
	Chain   []Package
}
