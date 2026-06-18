package mushroom

type Substrate interface {
	Digest(mushroomURL Hypha, data any, soil *Soil) (Mycelium, error)
	MushroomURL() string

	// Forage reads the raw nutrient content identified by mushroomURL from the
	// substrate's backing store and returns it. The returned value can be passed
	// directly to Digest. Implementations must be safe for concurrent use.
	Forage(mushroomURL Hypha) (any, error)

	// Sow writes nutrients (data) to the substrate's backing store at the
	// location identified by mushroomURL. data may be a string (written
	// verbatim) or any JSON-marshalable value. Implementations must be safe
	// for concurrent use.
	Sow(mushroomURL Hypha, data any) error
}
