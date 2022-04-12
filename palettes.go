package main

var c64palettes = map[string][16]colorInfo{
	"vice": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0xbc, 0x52, 0x41}},
		{3, RGB{0x8f, 0xef, 0xfb}},
		{4, RGB{0xb9, 0x56, 0xeb}},
		{5, RGB{0x7e, 0xdb, 0x40}},
		{6, RGB{0x55, 0x3f, 0xe4}},
		{7, RGB{0xff, 0xff, 0x77}},
		{8, RGB{0xc1, 0x7b, 0x1d}},
		{9, RGB{0x82, 0x63, 0x00}},
		{10, RGB{0xf4, 0x94, 0x86}},
		{11, RGB{0x72, 0x72, 0x72}},
		{12, RGB{0xa4, 0xa4, 0xa4}},
		{13, RGB{0xcd, 0xff, 0x98}},
		{14, RGB{0x9e, 0x8d, 0xff}},
		{15, RGB{0xd5, 0xd5, 0xd5}},
	},
	"vice old lum": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0xa9, 0x38, 0x26}},
		{3, RGB{0xae, 0xff, 0xff}},
		{4, RGB{0xdf, 0x82, 0xff}},
		{5, RGB{0x7e, 0xdb, 0x40}},
		{6, RGB{0x55, 0x3f, 0xe4}},
		{7, RGB{0xf7, 0xff, 0x6d}},
		{8, RGB{0xe7, 0xa4, 0x53}},
		{9, RGB{0x82, 0x63, 0x00}},
		{10, RGB{0xf4, 0x94, 0x86}},
		{11, RGB{0x5c, 0x5c, 0x5c}},
		{12, RGB{0xb0, 0xb0, 0xb0}},
		{13, RGB{0xc4, 0xff, 0x8f}},
		{14, RGB{0xaa, 0x99, 0xff}},
		{15, RGB{0xf2, 0xf2, 0xf2}},
	},
	"pepto": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0x68, 0x37, 0x2b}},
		{3, RGB{0x70, 0xa4, 0xb2}},
		{4, RGB{0x6f, 0x3d, 0x86}},
		{5, RGB{0x58, 0x8d, 0x43}},
		{6, RGB{0x35, 0x28, 0x79}},
		{7, RGB{0xb8, 0xc7, 0x6f}},
		{8, RGB{0x6f, 0x4f, 0x25}},
		{9, RGB{0x43, 0x39, 0x00}},
		{10, RGB{0x9a, 0x67, 0x59}},
		{11, RGB{0x44, 0x44, 0x44}},
		{12, RGB{0x6c, 0x6c, 0x6c}},
		{13, RGB{0x9a, 0xd2, 0x84}},
		{14, RGB{0x6c, 0x5e, 0xb5}},
		{15, RGB{0x95, 0x95, 0x95}},
	},
	"pantaloon": {
		{0, RGB{0, 0, 0}},
		{1, RGB{255, 255, 255}},
		{2, RGB{104, 55, 43}},
		{3, RGB{131, 240, 220}},
		{4, RGB{111, 61, 134}},
		{5, RGB{89, 205, 54}},
		{6, RGB{65, 55, 205}},
		{7, RGB{184, 199, 111}},
		{8, RGB{209, 127, 48}},
		{9, RGB{67, 57, 0}},
		{10, RGB{154, 103, 89}},
		{11, RGB{91, 91, 91}},
		{12, RGB{142, 142, 142}},
		{13, RGB{157, 255, 157}},
		{14, RGB{117, 161, 236}},
		{15, RGB{193, 193, 193}},
	},
	"archmage": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0x89, 0x40, 0x36}},
		{3, RGB{0x7a, 0xbf, 0xc7}},
		{4, RGB{0x8a, 0x46, 0xae}},
		{5, RGB{0x68, 0xa9, 0x41}},
		{6, RGB{0x3e, 0x31, 0xa2}},
		{7, RGB{0xd0, 0xdc, 0x71}},
		{8, RGB{0x90, 0x5f, 0x25}},
		{9, RGB{0x5c, 0x47, 0x00}},
		{10, RGB{0xbb, 0x77, 0x6d}},
		{11, RGB{0x55, 0x55, 0x55}},
		{12, RGB{0x80, 0x80, 0x80}},
		{13, RGB{0xac, 0xea, 0x88}},
		{14, RGB{0x7c, 0x70, 0xda}},
		{15, RGB{0xab, 0xab, 0xab}},
	},
	"electric cocillana": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0x8b, 0x1f, 0x00}},
		{3, RGB{0x6f, 0xdf, 0xb7}},
		{4, RGB{0xa7, 0x3b, 0x9f}},
		{5, RGB{0x4a, 0xb5, 0x10}},
		{6, RGB{0x08, 0x00, 0x94}},
		{7, RGB{0xf3, 0xeb, 0x5b}},
		{8, RGB{0xa5, 0x42, 0x00}},
		{9, RGB{0x63, 0x29, 0x18}},
		{10, RGB{0xcb, 0x7b, 0x6f}},
		{11, RGB{0x45, 0x44, 0x44}},
		{12, RGB{0x9f, 0x9f, 0x9f}},
		{13, RGB{0x94, 0xff, 0x94}},
		{14, RGB{0x4a, 0x94, 0xd6}},
		{15, RGB{0xbd, 0xbd, 0xbd}},
	},
	"ste": {
		{0, RGB{0x00, 0x00, 0x00}},
		{1, RGB{0xff, 0xff, 0xff}},
		{2, RGB{0xc8, 0x35, 0x35}},
		{3, RGB{0x83, 0xf0, 0xdc}},
		{4, RGB{0xcc, 0x59, 0xc6}},
		{5, RGB{0x59, 0xcd, 0x36}},
		{6, RGB{0x41, 0x37, 0xcd}},
		{7, RGB{0xf7, 0xee, 0x59}},
		{8, RGB{0xd1, 0x7f, 0x30}},
		{9, RGB{0x91, 0x5f, 0x33}},
		{10, RGB{0xf9, 0x9b, 0x97}},
		{11, RGB{0x5b, 0x5b, 0x5b}},
		{12, RGB{0x8e, 0x8e, 0x8e}},
		{13, RGB{0x9d, 0xff, 0x9d}},
		{14, RGB{0x75, 0xa1, 0xec}},
		{15, RGB{0xc1, 0xc1, 0xc1}},
	},
}
