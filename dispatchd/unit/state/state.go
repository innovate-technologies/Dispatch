package state

import "fmt"

// State represets the state of a unit
type State int

const (
	// Active means the unit is running
	Active State = iota
	// Dead means the unit stopped
	Dead
	// Starting means the unit is becoming active
	Starting
	// Destroy means that the holder of the unit should delete the unit
	Destroy
)

var nameStrings = [...]string{
	"active",
	"dead",
	"starting",
	"destroy"
}

var statePerInt = map[int]State{
	0: Active,
	1: Dead,
	2: Starting,
	3: Destroy
}

func (s State) String() string {
	return nameStrings[s]
}

// ForString sends back the state for the given string
func ForString(name string) State {
	for index, value := range nameStrings {
		if name == value {
			return statePerInt[index]
		}
	}
	fmt.Println(name)
	return Dead // if no match has been found
}
