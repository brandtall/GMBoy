package main

import (
	"fmt"
	"log"
	"os"
)

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
	IME            bool
	bus            Memory
	instructions   [256]Instruction
	cbInstructions [256]Instruction
	duration       int
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

func (c *CPU) ReadHL() uint16 {
	return uint16(c.H)<<8 | uint16(c.L)
}

func (c *CPU) ReadAF() uint16 {
	return uint16(c.A)<<8 | uint16(c.F)
}

func (c *CPU) WriteAF(value uint16) {
	high := uint8(value >> 8)
	low := uint8(value & 0xFF)
	c.A = high
	c.F = low
}

func (c *CPU) GetReg8(id int) uint8 {
	switch id {
	case 0:
		return c.B
	case 1:
		return c.C
	case 2:
		return c.D
	case 3:
		return c.E
	case 4:
		return c.H
	case 5:
		return c.L
	case 6:
		return c.bus.Read(c.ReadHL())
	case 7:
		return c.A
	}
	return 0xFF
}

func (c *CPU) WriteReg8(id int, v uint8) {
	switch id {
	case 0:
		c.B = v
	case 1:
		c.C = v
	case 2:
		c.D = v
	case 3:
		c.E = v
	case 4:
		c.H = v
	case 5:
		c.L = v
	case 6:
		c.bus.Write(c.ReadHL(), v)
	case 7:
		c.A = v
	}
}

func (c *CPU) SetReg16(id int, value uint16) {
	high := uint8(value >> 8)
	low := uint8(value & 0xFF)
	switch id {
	case 0:
		c.B = high
		c.C = low
	case 1:
		c.D = high
		c.E = low
	case 2:
		c.H = high
		c.L = low
	case 3:
		c.SP = value
	}
}

func (c *CPU) GetReg16(id int) uint16 {
	switch id {
	case 0:
		return uint16(c.B)<<8 | uint16(c.C)
	case 1:
		return uint16(c.D)<<8 | uint16(c.E)
	case 2:
		return uint16(c.H)<<8 | uint16(c.L)
	case 3:
		return c.SP
	}
	return 0
}

func (c *CPU) ExecuteALU(op int, val uint8) {
	valA := uint16(c.A)
	valB := uint16(val)

	switch op {
	case 0:
		sum := valA + valB
		halfCarry := (valA&0x0F)+(valB&0x0F) > 0x0F
		carry := sum > 0xFF
		zero := (sum & 0xFF) == 0x00
		c.A = uint8(sum)
		c.setFlags(zero, false, halfCarry, carry)
	case 1:
		carryIn := uint16(0)
		if (c.F & 0x10) != 0 {
			carryIn = 1
		}
		sum := valA + valB + carryIn
		halfCarry := (uint16(c.A&0x0F) + uint16(val&0x0F) + carryIn) > 0x0F
		carry := sum > 0xFF
		zero := (sum & 0xFF) == 0
		c.A = uint8(sum)
		c.setFlags(zero, false, halfCarry, carry)

	case 2:
		sub := valA - valB
		halfCarry := (c.A & 0x0F) < (val & 0x0F)
		carry := valA < valB
		zero := (sub & 0xFF) == 0
		c.A = uint8(sub)
		c.setFlags(zero, true, halfCarry, carry)

	case 3:
		carryIn := uint16(0)
		if (c.F & 0x10) != 0 {
			carryIn = 1
		}
		sub := valA - valB - carryIn
		halfCarry := int16(c.A&0xF)-int16(val&0xF)-int16(carryIn) < 0
		carry := int16(valA)-int16(valB)-int16(carryIn) < 0
		zero := (sub & 0xFF) == 0
		c.A = uint8(sub)
		c.setFlags(zero, true, halfCarry, carry)

	case 4:
		res := valA & valB
		zero := (res & 0xFF) == 0
		c.A = uint8(res)
		c.setFlags(zero, false, true, false)

	case 5:
		res := valA ^ valB
		zero := (res & 0xFF) == 0
		c.A = uint8(res)
		c.setFlags(zero, false, false, false)

	case 6:
		res := valA | valB
		zero := (res & 0xFF) == 0
		c.A = uint8(res)
		c.setFlags(zero, false, false, false)

	case 7:
		sub := valA - valB
		halfCarry := (c.A & 0x0F) < (val & 0x0F)
		carry := valA < valB
		zero := (sub & 0xFF) == 0
		c.setFlags(zero, true, halfCarry, carry)
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

func (c *CPU) executeCBShift(op int, val uint8) (uint8, uint8) {
	var res uint8
	var flagC bool

	switch op {
	case 0:
		bit7 := (val >> 7) & 0x01
		res = (val << 1) | bit7
		flagC = bit7 != 0

	case 1:
		bit0 := val & 0x01
		res = (val >> 1) | (bit0 << 7)
		flagC = bit0 != 0

	case 2:
		oldCarry := uint8(0)
		if (c.F & 0x10) != 0 {
			oldCarry = 1
		}
		res = (val << 1) | oldCarry
		flagC = (val & 0x80) != 0

	case 3:
		oldCarry := uint8(0)
		if (c.F & 0x10) != 0 {
			oldCarry = 0x80
		}
		res = (val >> 1) | oldCarry
		flagC = (val & 0x01) != 0

	case 4:
		flagC = (val & 0x80) != 0
		res = val << 1

	case 5:
		flagC = (val & 0x01) != 0
		msb := val & 0x80
		res = (val >> 1) | msb

	case 6:
		low := (val & 0x0F) << 4
		high := (val & 0xF0) >> 4
		res = low | high
		flagC = false

	case 7:
		flagC = (val & 0x01) != 0
		res = val >> 1
	}

	var newF uint8
	if res == 0 {
		newF |= 0x80
	}
	if flagC {
		newF |= 0x10
	}

	return res, newF
}

func main() {
	mmu := &MMU{}
	cpu := NewCPU(mmu)

	rom, err := os.ReadFile("cpu_instrs.gb")
	if err != nil {
		log.Fatal(err)
	}
	mmu.LoadCartridge(rom)

	cpu.PC = 0x0100

	fmt.Println("--- System Start ---")

	for {

		cycles := cpu.Step()

		_ = cycles
	}
}
