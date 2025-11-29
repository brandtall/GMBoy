package main

import (
	"fmt"
)

type OpcodeFunc func(*CPU)

type Instruction struct {
	Name   string
	Method OpcodeFunc
	Cycles int
}

type CPU struct {
	A              uint8
	F              uint8
	B              uint8
	C              uint8
	D              uint8
	E              uint8
	H              uint8
	L              uint8
	SP             uint16
	PC             uint16
	bus            Memory
	instructions   [256]Instruction
	cbInstructions [256]Instruction
	duration       int
}

type Memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, val uint8)
}

func NewCPU(bus Memory) *CPU {
	cpu := &CPU{
		A:   0x01,
		F:   0xB0,
		SP:  0xFFFE,
		PC:  0x100,
		bus: bus,
	}
	cpu.initInstructions()
	return cpu
}

func (c *CPU) initInstructions() {
	for i := range 256 {
		c.instructions[i] = Instruction{
			Name: "UNKNOWN",
			Method: func(cpu *CPU) {
				fmt.Printf("Unknown Opcode: 0x%X\n", cpu.bus.Read(cpu.PC-1))
			},
			Cycles: 0,
		}
		c.cbInstructions[i] = Instruction{
			Name: "UNKNOWN CB",
			Method: func(cpu *CPU) {
				fmt.Printf("Unknown CB Opcode: 0x%X\n", cpu.bus.Read(cpu.PC-1))
			},
			Cycles: 0,
		}
	}

	c.instructions[0x00] = Instruction{Name: "NOP", Cycles: 4, Method: func(c *CPU) {}}

	c.instructions[0xC3] = Instruction{
		Name: "JP nn", Cycles: 16, Method: func(c *CPU) {
			c.PC = c.fetchWord()
		},
	}

	c.instructions[0xCD] = Instruction{
		Name: "CALL nn", Cycles: 24, Method: func(c *CPU) {
			target := c.fetchWord()
			c.push(c.PC)
			c.PC = target
		},
	}

	c.instructions[0xC9] = Instruction{
		Name: "RET", Cycles: 16, Method: func(c *CPU) {
			c.PC = c.pop()
		},
	}

	c.instructions[0x80] = Instruction{
		Name: "ADD A,B", Cycles: 4, Method: func(c *CPU) {
			valA := uint16(c.A)
			valB := uint16(c.B)
			sum := valA + valB
			halfCarry := (c.A&0x0F)+(c.B&0x0F) > 0x0F
			carry := sum > 0xFF
			zero := (sum & 0xFF) == 0
			c.A = uint8(sum)
			c.setFlags(zero, false, halfCarry, carry)
		},
	}

	c.instructions[0x90] = Instruction{
		Name: "SUB A,B", Cycles: 4, Method: func(c *CPU) {
			valA := uint16(c.A)
			valB := uint16(c.B)
			sub := valA - valB
			halfCarry := (c.A & 0x0F) < (c.B & 0x0F)
			carry := valA < valB
			zero := (sub & 0xFF) == 0
			c.A = uint8(sub)
			c.setFlags(zero, true, halfCarry, carry)
		},
	}

	c.instructions[0xCB] = Instruction{
		Name: "PREFIX CB", Cycles: 0, Method: func(c *CPU) {
			cbOpcode := c.fetchByte()
			ins := c.cbInstructions[cbOpcode]
			ins.Method(c)
			c.duration += ins.Cycles
		},
	}
}

func (c *CPU) Step() int {
	c.duration = 0
	opcode := c.fetchByte()
	ins := c.instructions[opcode]
	ins.Method(c)
	return ins.Cycles + c.duration
}

func (c *CPU) fetchByte() uint8 {
	opcode := c.bus.Read(c.PC)
	c.PC++
	return opcode
}

func (c *CPU) fetchWord() uint16 {
	low := c.fetchByte()
	high := c.fetchByte()
	return uint16(high)<<8 | uint16(low)
}

func (c *CPU) push(val uint16) {
	c.SP--
	c.bus.Write(c.SP, uint8(val>>8))
	c.SP--
	c.bus.Write(c.SP, uint8(val))
}

func (c *CPU) pop() uint16 {
	l := c.bus.Read(c.SP)
	c.SP++
	h := c.bus.Read(c.SP)
	c.SP++
	return uint16(h)<<8 | uint16(l)
}

func (c *CPU) setFlags(z, n, h, cy bool) {
	var f uint8
	if z {
		f |= 0x80
	}
	if n {
		f |= 0x40
	}
	if h {
		f |= 0x20
	}
	if cy {
		f |= 0x10
	}
	c.F = f
}

type GBMemory struct{ data [65536]byte }

func (m *GBMemory) Read(a uint16) uint8      { return m.data[a] }
func (m *GBMemory) Write(a uint16, v uint8)  { m.data[a] = v }
func (m *GBMemory) LoadCartridge(rom []byte) { copy(m.data[0:], rom) }

func main() {}
