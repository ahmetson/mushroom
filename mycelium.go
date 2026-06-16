package mushroom

type Mycelium interface {
	Link(string) (string, error)
	Spore(string) (any, error)
	Fruit(any) (any, error)
	Mineralize() (any, error)
	MushroomURL() string
	Soil() *Soil
	Substrate() *Substrate
}
