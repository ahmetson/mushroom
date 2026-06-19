package mushroom

type Mycelium interface {
	Link(mushroomURL string) (string, error) // Converts the url into a full path with validation
	Spore(mushroomURL string) (any, error)   // Dereferences the URL, returning its value; germinates unknown mycelia on demand
	Fruit(any) (any, error)                  // Traverses a value, finds any links inside it, and evaluates those links when needed
	Mineralize() (any, error)                // Convert back the mycelium into its raw format
	MushroomURL() string                     // Returns the absolute link to the mycelium
	MyceliumURL() Hypha                      // Returns the absolute link hypha of the mycelium
	Soil() *Soil
	Substrate() *Substrate
}
