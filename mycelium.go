package mushroom

type Mycelium interface {
	Link(mushroomURL string) (string, error)
	Spore(mushroomURL string) (any, error)
	Fruit(any) (any, error)
	Mineralize() (any, error)
	MushroomURL() string
	Soil() *Soil
	Substrate() *Substrate
}
