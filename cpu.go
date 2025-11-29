package main

import (
	"fmt"
	"time"
)

type CPU struct {
	A   uint8
	F   uint8
	B   uint8
	C   uint8
	D   uint8
	E   uint8
	H   uint8
	L   uint8
	SP  uint16
	PC  uint16
	bus Memory
}

type Memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, val uint8)
}

func NewCPU(bus Memory) *CPU {
	return &CPU{
		A:   0x01,
		F:   0xB0,
		SP:  0xFFFE,
		PC:  0x100,
		bus: bus,
	}
}

func (c *CPU) ReadAF() uint16 {
	return uint16(c.A)<<8 | uint16(c.F)
}

func (c *CPU) WriteAF(value uint16) {
	c.A = uint8(value >> 8)
	c.F = uint8(value & 0xF0)
}

func (c *CPU) Step() int {
	opcode := c.fetchByte()

	switch opcode {
	case 0x00:
		return 4
	case 0xC3:
		c.PC = c.fetchWord()
		fmt.Printf("Jumped to 0x%X\n", c.PC)
		return 16
	default:
		fmt.Printf("Unkown Opcode: 0x%X\n", opcode)
		return 0
	}
}

func (c *CPU) fetchByte() uint8 {
	opcode := c.bus.Read(c.PC)
	c.PC++
	return opcode
}

func (c *CPU) fetchWord() uint16 {
	lowByte := c.fetchByte()
	highByte := c.fetchByte()
	return uint16(highByte)<<8 | uint16(lowByte)
}

func (c *CPU) TickComponents(cycles int) {
	fmt.Printf(">> Advancing PPU by %d cycles\n", cycles)
}

type GBMemory struct {
	data [65536]byte
}

func (m *GBMemory) Read(address uint16) uint8 {
	return m.data[address]
}

func (m *GBMemory) Write(address uint16, value uint8) {
	m.data[address] = value
}

func (m *GBMemory) LoadCartridge(rom []byte) {
	copy(m.data[0:], rom)
}

func main() {
	memory := GBMemory{}
	cpu := NewCPU(&memory)
	fakeRom := []byte{0x00, 0x00, 0xC3, 0xC3, 0x00, 0x01}
	memory.LoadCartridge(fakeRom)

	cpu.PC = 0x0000

	for {
		stepCycles := cpu.Step()
		cpu.TickComponents(stepCycles)
		fmt.Printf("PC: 0x%04X | Cycles: %d\n", cpu.PC, stepCycles)
		time.Sleep(100 * time.Millisecond)
	}
}
