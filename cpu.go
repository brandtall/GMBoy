package main

import(
	"fmt"
)

type CPU struct {
	A  uint8
	F  uint8
	B  uint8
	C  uint8
	D  uint8
	E  uint8
	H  uint8
	L  uint8
	SP uint16
	PC uint16
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

func (c *CPU) Step() {
	instruction := c.bus.Read(c.PC)
	c.PC += 1
	fmt.Printf("Current instruction is %d", instruction)
}

type GBMemory struct{
	data [65536]byte
}

func (m *GBMemory) Read(address uint16) uint8 {
	return m.data[address]
}
func (m *GBMemory) Write(address uint16, value uint8) {
	m.data[address] = value
}



func main() {}
