package inventory

// Equipment represents one physical item in the laboratory inventory.
type Equipment struct {
	ID   string
	Name string
}

// InitialEquipment returns the equipment catalog installed on first startup.
func InitialEquipment() []Equipment {
	return []Equipment{
		{ID: "MIC-01", Name: "Microscopio digital"},
		{ID: "MIC-02", Name: "Microscopio digital"},
		{ID: "OSC-01", Name: "Osciloscopio"},
		{ID: "MUL-01", Name: "Multímetro"},
		{ID: "FUE-01", Name: "Fuente de alimentación"},
	}
}
