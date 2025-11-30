package main

type Memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, val uint8)
}

type MMU struct {
	rom  [0x8000]byte
	vram [0x2000]byte
	eram [0x2000]byte
	wram [0x2000]byte
	oam  [0xA0]byte
	io   [0x80]byte
	hram [0x7F]byte
	ie   byte
}

func (m *MMU) Read(a uint16) uint8 {
	switch {
	case a < 0x8000:

		return m.rom[a]

	case a >= 0x8000 && a < 0xA000:

		return m.vram[a-0x8000]

	case a >= 0xA000 && a < 0xC000:

		return m.eram[a-0xA000]

	case a >= 0xC000 && a < 0xE000:

		return m.wram[a-0xC000]

	case a >= 0xE000 && a < 0xFE00:

		return 0xFF

	case a >= 0xFE00 && a < 0xFEA0:

		return m.oam[a-0xFE00]

	case a >= 0xFF00 && a < 0xFF80:

		return m.io[a-0xFF00]

	case a >= 0xFF80 && a < 0xFFFF:

		return m.hram[a-0xFF80]

	case a == 0xFFFF:

		return m.ie

	default:

		return 0xFF
	}
}

func (m *MMU) Write(a uint16, v uint8) {
	switch {
	case a < 0x8000:

		return

	case a >= 0x8000 && a < 0xA000:
		m.vram[a-0x8000] = v

	case a >= 0xA000 && a < 0xC000:
		m.eram[a-0xA000] = v

	case a >= 0xC000 && a < 0xE000:
		m.wram[a-0xC000] = v

	case a >= 0xFE00 && a < 0xFEA0:
		m.oam[a-0xFE00] = v

	case a >= 0xFF00 && a < 0xFF80:
		m.io[a-0xFF00] = v

	case a >= 0xFF80 && a < 0xFFFF:
		m.hram[a-0xFF80] = v

	case a == 0xFFFF:
		m.ie = v
	}
}

func (m *MMU) LoadCartridge(rom []byte) {
	copy(m.rom[:], rom)
}
