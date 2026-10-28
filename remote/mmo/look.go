package mmo

// How a character looks: a humanoid with alien touches. Every field is an index into
// the lists below, so the server can check them and any game can draw anyone.

type Look struct {
	Skin    int `json:"skin"`
	Eyes    int `json:"eyes"`
	Feature int `json:"feature"` // antennae, ears, horns...
	Hair    int `json:"hair"`    // style
	HairCol int `json:"hairCol"`
	Shirt   int `json:"shirt"`
	Pants   int `json:"pants"`
}

var (
	SkinTones = [][3]int{
		{255, 214, 180}, {214, 160, 118}, {140, 94, 64}, // human
		{124, 98, 255}, {150, 196, 255}, {110, 204, 150}, {72, 168, 186}, {236, 140, 200}, {200, 210, 220}, // alien
	}
	EyeColours   = [][3]int{{30, 26, 46}, {70, 140, 255}, {60, 190, 110}, {255, 200, 60}, {240, 70, 80}, {190, 110, 255}, {120, 240, 255}}
	HairColours  = [][3]int{{40, 32, 40}, {110, 70, 40}, {230, 196, 110}, {200, 80, 50}, {240, 240, 250}, {124, 98, 255}, {70, 200, 170}, {255, 130, 190}}
	ClothColours = [][3]int{{124, 98, 255}, {60, 120, 220}, {70, 160, 90}, {200, 70, 70}, {230, 170, 60}, {60, 60, 80}, {230, 230, 240}, {150, 90, 60}, {50, 170, 170}}
	HairStyles   = []string{"None", "Short", "Long", "Spiky", "Bun", "Crest"}
	Features     = []string{"None", "Antennae", "Pointed ears", "Horns", "Head fin"}
)

// Valid keeps every field in range
func (l *Look) Valid() {
	clamp := func(v *int, n int) { *v = max(0, min(n-1, *v)) }
	clamp(&l.Skin, len(SkinTones))
	clamp(&l.Eyes, len(EyeColours))
	clamp(&l.Feature, len(Features))
	clamp(&l.Hair, len(HairStyles))
	clamp(&l.HairCol, len(HairColours))
	clamp(&l.Shirt, len(ClothColours))
	clamp(&l.Pants, len(ClothColours))
}

// The look a new character starts with (the violet of the mote, with antennae)
var DefaultLook = Look{Skin: 3, Eyes: 6, Feature: 1, Hair: 1, HairCol: 0, Shirt: 4, Pants: 5}
