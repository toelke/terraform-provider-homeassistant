package client

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		// From test_slugify in Home Assistant's tests/util/test_init.py.
		{"T-!@#$!#@$!$est", "t_est"},
		{"Test More", "test_more"},
		{"Test_(More)", "test_more"},
		{"Tèst_Mörê", "test_more"},
		{"B8:27:EB:00:00:00", "b8_27_eb_00_00_00"},
		{"test.com", "test_com"},
		{"greg_phone - exp_wayp1", "greg_phone_exp_wayp1"},
		{"We are, we are, a... Test Calendar", "we_are_we_are_a_test_calendar"},
		{"Tèst_äöüß_ÄÖÜ", "test_aouss_aou"},
		{"影師嗎", "ying_shi_ma"},
		{"けいふぉんと", "keihuonto"},
		{"$247", "247"},
		{"123", "123"},
		{"$$$", "unknown"},
		{"", ""},

		// Further vectors, with the output of HA 2026.9.4's slugify.
		{"Living Room", "living_room"},
		{"Bob's Lamp", "bob_s_lamp"},
		{"1,000 Watts", "1000_watts"},
		{"a, b", "a_b"},
		{"Tom &amp; Jerry", "tom_jerry"},
		{"caf&eacute;", "cafe"},
		{"&#65;&#x42;c", "abc"},
		{"&#9999999999;&#65;", "9999999999_65"},
		{"&bogus; x", "bogus_x"},
		{"Ⅻ ① ﬁ", "xii_1_fi"},
		{"Привет мир", "privet_mir"},
		{"שלום", "shlvm"},
		{"Γειά σου", "geia_sou"},
		{"🏠 Home", "home"},
		{"½ Bath", "1_2_bath"},
		{"  padded  ", "padded"},
		{"İstanbul", "istanbul"},
		{"Straße", "strasse"},
		{"Crème brûlée", "creme_brulee"},
		{"日本語ラベル", "ri_ben_yu_raheru"},
		{"xאy", "xy"},
		{"a͐b", "a_b"},
	}
	for _, tt := range tests {
		if got := Slugify(tt.in); got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
