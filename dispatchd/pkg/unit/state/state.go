package state

// State represets the state of a unit
type State int

const (
	// Active means the unit is running
	Active State = iota
	// Dead means the unit stopped
	Dead
	// Starting means the unit is becoming active
	Starting
)

var nameStrings = [...]string{
	"active",
	"dead",
	"starting",
}

var statePerInt = map[int]State{
	0: Active,
	1: Dead,
	2: Starting,
}

func (s State) String() string {
	return nameStrings[s]
}

// StateForString sends back the state for the given string
func ForString(name string) State {
	for index, value := range nameStrings {
		if name == value {
			return statePerInt[index]
		}
	}
	return Dead // if no match has been found
}
