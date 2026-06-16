package mushroom

type Substrate interface {
	Digest(mushroomURL string, data any, soil *Soil) (Mycelium, error)
	MushroomURL() string
}
