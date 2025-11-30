package main

import (
	"fmt"
	"os"
)

type OpcodeFunc func(*CPU)

type Instruction struct {
	Name   string
	Method OpcodeFunc
	Cycles int
}

func (c *CPU) initInstructions() {
	for i := range 256 {
		c.instructions[i] = Instruction{
			Name: "UNKNOWN",
			Method: func(cpu *CPU) {
				fmt.Printf("Unknown Opcode: 0x%X\n", cpu.bus.Read(cpu.PC-1))
				os.Exit(1)
			},
			Cycles: 0,
		}
		c.cbInstructions[i] = Instruction{
			Name: "UNKNOWN CB",
			Method: func(cpu *CPU) {
				fmt.Printf("Unknown CB Opcode: 0x%X\n", cpu.bus.Read(cpu.PC-1))
				os.Exit(1)
			},
			Cycles: 0,
		}
	}

	for opcode := 0x40; opcode < 0x80; opcode++ {
		if opcode == 0x76 {
			continue
		}
		op := opcode
		destID := (op >> 3) & 0x07
		srcID := op & 0x07

		cycles := 4
		if destID == 6 || srcID == 6 {
			cycles = 8
		}

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("LD %d,%d", destID, srcID),
			Method: func(cpu *CPU) {
				val := cpu.GetReg8(srcID)
				cpu.WriteReg8(destID, val)
			},
			Cycles: cycles,
		}
	}

	for opcode := 0x80; opcode < 0xC0; opcode++ {
		op := opcode
		aluOp := (op >> 3) & 0x07
		srcReg := op & 0x07

		cycles := 4
		if srcReg == 6 {
			cycles = 8
		}

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("ALU %d,%d", aluOp, srcReg),
			Method: func(cpu *CPU) {
				valA := uint16(cpu.A)
				valB := uint16(cpu.GetReg8(srcReg))

				switch aluOp {
				case 0:
					sum := valA + valB

					halfCarry := (cpu.A&0x0F)+(uint8(valB)&0x0F) > 0x0F
					carry := sum > 0xFF
					zero := (sum & 0xFF) == 0
					cpu.A = uint8(sum)
					cpu.setFlags(zero, false, halfCarry, carry)

				case 1:
					carryIn := uint16(0)
					if (cpu.F & 0x10) != 0 {
						carryIn = 1
					}

					sum := valA + valB + carryIn

					halfCarry := (uint16(cpu.A&0x0F) + uint16(uint8(valB)&0x0F) + carryIn) > 0x0F
					carry := sum > 0xFF
					zero := (sum & 0xFF) == 0
					cpu.A = uint8(sum)
					cpu.setFlags(zero, false, halfCarry, carry)

				case 2:
					sub := valA - valB
					halfCarry := (cpu.A & 0x0F) < (uint8(valB) & 0x0F)
					carry := valA < valB
					zero := (sub & 0xFF) == 0
					cpu.A = uint8(sub)
					cpu.setFlags(zero, true, halfCarry, carry)

				case 3:
					carryIn := uint16(0)
					if (cpu.F & 0x10) != 0 {
						carryIn = 1
					}

					sub := valA - valB - carryIn

					halfCarry := int16(cpu.A&0xF)-int16(valB&0xF)-int16(carryIn) < 0
					carry := int16(valA)-int16(valB)-int16(carryIn) < 0

					zero := (sub & 0xFF) == 0
					cpu.A = uint8(sub)
					cpu.setFlags(zero, true, halfCarry, carry)

				case 4:
					res := valA & valB
					zero := (res & 0xFF) == 0
					cpu.A = uint8(res)

					cpu.setFlags(zero, false, true, false)

				case 5:
					res := valA ^ valB
					zero := (res & 0xFF) == 0
					cpu.A = uint8(res)
					cpu.setFlags(zero, false, false, false)

				case 6:
					res := valA | valB
					zero := (res & 0xFF) == 0
					cpu.A = uint8(res)
					cpu.setFlags(zero, false, false, false)

				case 7:

					sub := valA - valB
					halfCarry := (cpu.A & 0x0F) < (uint8(valB) & 0x0F)
					carry := valA < valB
					zero := (sub & 0xFF) == 0

					cpu.setFlags(zero, true, halfCarry, carry)
				}
			},
			Cycles: cycles,
		}
	}

	for regID := range 8 {

		cycles := 4
		if regID == 6 {
			cycles = 12
		}
		incID := (regID << 3) | 0x04
		c.instructions[incID] = Instruction{
			Name: "INC %d",
			Method: func(c *CPU) {
				val := c.GetReg8(regID)
				res := val + 1
				c.WriteReg8(regID, res)
				halfCarry := (val & 0x0F) == 0x0F
				carry := c.F & 0x10
				zero := res == 0x00
				c.setFlags(zero, false, halfCarry, carry != 0)
			},
			Cycles: cycles,
		}
		decID := (regID << 3) | 0x05
		c.instructions[decID] = Instruction{
			Name: "DEC %d",
			Method: func(c *CPU) {
				val := c.GetReg8(regID)
				res := val - 1
				c.WriteReg8(regID, res)
				halfCarry := (val & 0x0F) == 0x00
				carry := c.F & 0x10
				zero := res == 0x00
				c.setFlags(zero, true, halfCarry, carry == 1)
			},
			Cycles: cycles,
		}
	}
	for i := range 8 {
		regID := i

		cycles := 8
		if regID == 6 {
			cycles = 12
		}

		op := (regID << 3) | 0x06
		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("LD %d, n", regID),
			Method: func(c *CPU) {
				val := c.fetchByte()
				c.WriteReg8(regID, val)
			},
			Cycles: cycles,
		}
	}

	for i := range 8 {
		target := uint16(i * 8)
		op := 0xC7 + (i * 8)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("RST %02XH", target),
			Method: func(c *CPU) {
				c.push(c.PC)
				c.PC = target
			},
			Cycles: 16,
		}
	}

	for i := range 4 {
		regID := i
		op := (regID << 4) | 0x01

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("LD r16, nn (%d)", regID),
			Method: func(c *CPU) {
				val := c.fetchWord()
				c.SetReg16(regID, val)
			},
			Cycles: 12,
		}
	}
	for i := range 4 {
		regID := i
		op := 0xC5 + (regID * 16)

		if i == 3 {
			c.instructions[op] = Instruction{
				Name: "PUSH rr (3)",
				Method: func(c *CPU) {
					c.push(c.ReadAF())
				},
				Cycles: 16,
			}
			continue
		}
		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("PUSH rr (%d)", regID),
			Method: func(c *CPU) {
				val := c.GetReg16(regID)
				c.push(val)
			},
			Cycles: 16,
		}
	}
	for i := range 4 {
		regID := i
		op := 0xC1 + (regID * 16)

		if i == 3 {
			c.instructions[op] = Instruction{
				Name: "POP rr (3)",
				Method: func(c *CPU) {
					c.WriteAF(c.pop())
				},
				Cycles: 12,
			}
			continue
		}
		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("POP rr (%d)", regID),
			Method: func(c *CPU) {
				val := c.pop()
				c.SetReg16(regID, val)
			},
			Cycles: 12,
		}
	}

	for i := range 4 {
		regID := i

		incOp := 0x03 + (i * 16)
		c.instructions[incOp] = Instruction{
			Name: fmt.Sprintf("INC r16(%d)", regID),
			Method: func(c *CPU) {
				val := c.GetReg16(regID)
				c.SetReg16(regID, val+1)
			},
			Cycles: 8,
		}

		decOp := 0x0B + (i * 16)
		c.instructions[decOp] = Instruction{
			Name: fmt.Sprintf("DEC r16(%d)", regID),
			Method: func(c *CPU) {
				val := c.GetReg16(regID)
				c.SetReg16(regID, val-1)
			},
			Cycles: 8,
		}
	}

	c.instructions[0x00] = Instruction{Name: "NOP", Cycles: 4, Method: func(c *CPU) {}}

	c.instructions[0xF3] = Instruction{
		Name: "DI",
		Method: func(*CPU) {
			c.IME = false
		},
		Cycles: 4,
	}
	c.instructions[0xFB] = Instruction{
		Name: "EI",
		Method: func(*CPU) {
			c.IME = true
		},
		Cycles: 4,
	}
	c.instructions[0xEA] = Instruction{
		Name: "LD (nn), A",
		Method: func(cpu *CPU) {
			addr := cpu.fetchWord()
			cpu.bus.Write(addr, cpu.A)
		},
		Cycles: 16,
	}

	c.instructions[0xFA] = Instruction{
		Name: "LD A, (nn)",
		Method: func(cpu *CPU) {
			addr := cpu.fetchWord()
			cpu.A = cpu.bus.Read(addr)
		},
		Cycles: 16,
	}
	c.instructions[0xE0] = Instruction{
		Name: "LDH n, A",
		Method: func(cpu *CPU) {
			addr := cpu.fetchByte()
			cpu.bus.Write(0xFF00+uint16(addr), c.A)
		},
		Cycles: 12,
	}
	c.instructions[0xF0] = Instruction{
		Name: "LDH A, n",
		Method: func(cpu *CPU) {
			addr := cpu.fetchByte()
			val := cpu.bus.Read(0xFF00 + uint16(addr))
			c.A = val
		},
		Cycles: 12,
	}
	c.instructions[0xC3] = Instruction{
		Name: "JP nn", Cycles: 16, Method: func(c *CPU) {
			c.PC = c.fetchWord()
		},
	}

	c.instructions[0x18] = Instruction{
		Name:   "JR n",
		Cycles: 12,
		Method: func(c *CPU) {
			offset := int8(c.fetchByte())
			c.PC = uint16(int32(c.PC) + int32(offset))
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

	c.cbInstructions[0x37] = Instruction{
		Name: "SWAP A", Method: func(c *CPU) {
			val := c.A
			upper := (val & 0xF0) >> 4
			lower := (val & 0x0F) << 4
			res := lower | upper

			c.A = res

			c.setFlags(res == 0, false, false, false)
		}, Cycles: 16,
	}

	c.instructions[0x22] = Instruction{
		Name: "LDI (HL), A",
		Method: func(c *CPU) {
			addr := c.GetReg16(2)
			c.bus.Write(addr, c.A)
			c.SetReg16(2, addr+1)
		},
		Cycles: 8,
	}

	c.instructions[0x2A] = Instruction{
		Name: "LDI A, (HL)",
		Method: func(c *CPU) {
			addr := c.GetReg16(2)
			c.A = c.bus.Read(addr)
			c.SetReg16(2, addr+1)
		},
		Cycles: 8,
	}

	c.instructions[0x32] = Instruction{
		Name: "LDD (HL), A",
		Method: func(c *CPU) {
			addr := c.GetReg16(2)
			c.bus.Write(addr, c.A)
			c.SetReg16(2, addr-1)
		},
		Cycles: 8,
	}

	c.instructions[0x3A] = Instruction{
		Name: "LDD A, (HL)",
		Method: func(c *CPU) {
			addr := c.GetReg16(2)
			c.A = c.bus.Read(addr)
			c.SetReg16(2, addr-1)
		},
		Cycles: 8,
	}

	c.instructions[0x20] = Instruction{
		Name: "JR NZ, n",
		Method: func(c *CPU) {
			offset := int8(c.fetchByte())
			if (c.F & 0x80) == 0 {
				c.PC = uint16(int32(c.PC) + int32(offset))
				c.duration += 4
			}
		},
		Cycles: 8,
	}

	c.instructions[0x28] = Instruction{
		Name: "JR Z, n",
		Method: func(c *CPU) {
			offset := int8(c.fetchByte())
			if (c.F & 0x80) != 0 {
				c.PC = uint16(int32(c.PC) + int32(offset))
				c.duration += 4
			}
		},
		Cycles: 8,
	}

	c.instructions[0x30] = Instruction{
		Name: "JR NC, n",
		Method: func(c *CPU) {
			offset := int8(c.fetchByte())
			if (c.F & 0x10) == 0 {
				c.PC = uint16(int32(c.PC) + int32(offset))
				c.duration += 4
			}
		},
		Cycles: 8,
	}

	c.instructions[0x38] = Instruction{
		Name: "JR C, n",
		Method: func(c *CPU) {
			offset := int8(c.fetchByte())
			if (c.F & 0x10) != 0 {
				c.PC = uint16(int32(c.PC) + int32(offset))
				c.duration += 4
			}
		},
		Cycles: 8,
	}
}
