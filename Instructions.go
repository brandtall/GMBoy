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
				val := cpu.GetReg8(srcReg)
				cpu.ExecuteALU(aluOp, val)
			},
			Cycles: cycles,
		}
	}

	for i := range 8 {
		aluOp := i
		op := 0xC6 + (i * 8)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("ALU A, n (%d)", aluOp),
			Method: func(c *CPU) {
				val := c.fetchByte()
				c.ExecuteALU(aluOp, val)
			},
			Cycles: 8,
		}
	}

	for i := range 2 {
		regID := i

		storeOp := 0x02 + (i * 16)
		c.instructions[storeOp] = Instruction{
			Name: fmt.Sprintf("LD (r16 %d), A", regID),
			Method: func(c *CPU) {
				addr := c.GetReg16(regID)
				c.bus.Write(addr, c.A)
			},
			Cycles: 8,
		}

		loadOp := 0x0A + (i * 16)
		c.instructions[loadOp] = Instruction{
			Name: fmt.Sprintf("LD A, (r16 %d)", regID),
			Method: func(c *CPU) {
				addr := c.GetReg16(regID)
				c.A = c.bus.Read(addr)
			},
			Cycles: 8,
		}
	}

	for i := range 4 {
		condition := i
		op := 0xC4 + (i * 8)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("CALL cc, nn (%d)", condition),
			Method: func(c *CPU) {
				target := c.fetchWord()

				shouldCall := false
				switch condition {
				case 0:
					shouldCall = (c.F & 0x80) == 0
				case 1:
					shouldCall = (c.F & 0x80) != 0
				case 2:
					shouldCall = (c.F & 0x10) == 0
				case 3:
					shouldCall = (c.F & 0x10) != 0
				}

				if shouldCall {
					c.push(c.PC)
					c.PC = target
					c.duration += 12
				}
			},
			Cycles: 12,
		}
	}

	for i := range 4 {
		condition := i
		op := 0xC2 + (i * 8)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("JP cc, nn (%d)", condition),
			Method: func(c *CPU) {
				target := c.fetchWord()

				shouldJump := false
				switch condition {
				case 0:
					shouldJump = (c.F & 0x80) == 0
				case 1:
					shouldJump = (c.F & 0x80) != 0
				case 2:
					shouldJump = (c.F & 0x10) == 0
				case 3:
					shouldJump = (c.F & 0x10) != 0
				}

				if shouldJump {
					c.PC = target
					c.duration += 4
				}
			},
			Cycles: 12,
		}
	}

	for i := range 4 {
		condition := i
		op := 0xC0 + (i * 8)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("RET cc (%d)", condition),
			Method: func(c *CPU) {
				shouldRet := false
				switch condition {
				case 0:
					shouldRet = (c.F & 0x80) == 0
				case 1:
					shouldRet = (c.F & 0x80) != 0
				case 2:
					shouldRet = (c.F & 0x10) == 0
				case 3:
					shouldRet = (c.F & 0x10) != 0
				}

				if shouldRet {
					c.PC = c.pop()
					c.duration += 12
				}
			},
			Cycles: 8,
		}
	}

	for i := range 4 {
		regID := i
		op := 0x09 + (i * 16)

		c.instructions[op] = Instruction{
			Name: fmt.Sprintf("ADD HL, r16(%d)", regID),
			Method: func(c *CPU) {
				valHL := uint32(c.GetReg16(2))
				valRR := uint32(c.GetReg16(regID))

				sum := valHL + valRR

				currentZ := (c.F & 0x80) != 0

				halfCarry := (valHL&0xFFF)+(valRR&0xFFF) > 0xFFF
				carry := sum > 0xFFFF

				c.SetReg16(2, uint16(sum))
				c.setFlags(currentZ, false, halfCarry, carry)
			},
			Cycles: 8,
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
				c.setFlags(zero, true, halfCarry, carry != 0)
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

	c.instructions[0xE9] = Instruction{
		Name: "JP (HL)",
		Method: func(c *CPU) {
			c.PC = c.GetReg16(2)
		},
		Cycles: 4,
	}

	c.instructions[0xF9] = Instruction{
		Name: "LD SP, HL",
		Method: func(c *CPU) {
			c.SP = c.GetReg16(2)
		},
		Cycles: 8,
	}

	c.instructions[0x07] = Instruction{
		Name: "RLCA",
		Method: func(c *CPU) {
			val := c.A
			bit7 := (val >> 7) & 0x01
			res := (val << 1) | bit7
			c.A = res
			c.setFlags(false, false, false, bit7 != 0)
		},
		Cycles: 4,
	}

	c.instructions[0x17] = Instruction{
		Name: "RLA",
		Method: func(c *CPU) {
			val := c.A
			bit7 := (val >> 7) & 0x01
			oldCarry := uint8(0)
			if (c.F & 0x10) != 0 {
				oldCarry = 1
			}
			res := (val << 1) | oldCarry
			c.A = res
			c.setFlags(false, false, false, bit7 != 0)
		},
		Cycles: 4,
	}

	c.instructions[0x0F] = Instruction{
		Name: "RRCA",
		Method: func(c *CPU) {
			val := c.A
			bit0 := val & 0x01
			res := (val >> 1) | (bit0 << 7)
			c.A = res
			c.setFlags(false, false, false, bit0 != 0)
		},
		Cycles: 4,
	}

	c.instructions[0x1F] = Instruction{
		Name: "RRA",
		Method: func(c *CPU) {
			val := c.A
			bit0 := val & 0x01
			oldCarry := uint8(0)
			if (c.F & 0x10) != 0 {
				oldCarry = 128
			}
			res := (val >> 1) | oldCarry
			c.A = res
			c.setFlags(false, false, false, bit0 != 0)
		},
		Cycles: 4,
	}

	c.instructions[0xF8] = Instruction{
		Name: "LD HL, SP+n",
		Method: func(c *CPU) {
			signedByte := int8(c.fetchByte())

			val := uint16(int32(c.SP) + int32(signedByte))

			c.SetReg16(2, val)

			rawOffset := uint16(uint8(signedByte))

			halfCarry := (c.SP&0x0F)+(rawOffset&0x0F) > 0x0F
			carry := (c.SP&0xFF)+(rawOffset&0xFF) > 0xFF

			c.setFlags(false, false, halfCarry, carry)
		},
		Cycles: 12,
	}

	c.instructions[0x2F] = Instruction{
		Name: "CPL",
		Method: func(c *CPU) {
			c.A = ^c.A

			z := (c.F & 0x80) != 0
			cy := (c.F & 0x10) != 0
			c.setFlags(z, true, true, cy)
		},
		Cycles: 4,
	}

	c.instructions[0x37] = Instruction{
		Name: "SCF",
		Method: func(c *CPU) {
			z := (c.F & 0x80) != 0
			c.setFlags(z, false, false, true)
		},
		Cycles: 4,
	}

	c.instructions[0x3F] = Instruction{
		Name: "CCF",
		Method: func(c *CPU) {
			z := (c.F & 0x80) != 0
			cy := (c.F & 0x10) != 0
			c.setFlags(z, false, false, !cy)
		},
		Cycles: 4,
	}

	for i := range 64 {
		op := i
		rotType := (op >> 3) & 0x07
		regID := op & 0x07

		cycles := 8
		if regID == 6 {
			cycles = 16
		}

		c.cbInstructions[op] = Instruction{
			Name: fmt.Sprintf("CB ROT %d, %d", rotType, regID),
			Method: func(c *CPU) {
				val := c.GetReg8(regID)
				res, newF := c.executeCBShift(rotType, val)
				c.WriteReg8(regID, res)
				c.F = newF
			},
			Cycles: cycles,
		}

		c.instructions[0x10] = Instruction{
			Name: "STOP",
			Method: func(c *CPU) {
				c.fetchByte()
			},
			Cycles: 4,
		}

		c.instructions[0x76] = Instruction{
			Name: "HALT",
			Method: func(c *CPU) {
			},
			Cycles: 4,
		}

		c.instructions[0x08] = Instruction{
			Name: "LD (nn), SP",
			Method: func(c *CPU) {
				addr := c.fetchWord()
				low := uint8(c.SP & 0xFF)
				high := uint8(c.SP >> 8)
				c.bus.Write(addr, low)
				c.bus.Write(addr+1, high)
			},
			Cycles: 20,
		}

		c.instructions[0xE8] = Instruction{
			Name: "ADD SP, n",
			Method: func(c *CPU) {
				signedByte := int8(c.fetchByte())

				rawOffset := uint16(uint8(signedByte))
				halfCarry := (c.SP&0x0F)+(rawOffset&0x0F) > 0x0F
				carry := (c.SP&0xFF)+(rawOffset&0xFF) > 0xFF

				c.SP = uint16(int32(c.SP) + int32(signedByte))
				c.setFlags(false, false, halfCarry, carry)
			},
			Cycles: 16,
		}

		c.instructions[0x27] = Instruction{
			Name: "DAA",
			Method: func(c *CPU) {
				val := int(c.A)
				correction := 0
				flagC := (c.F & 0x10) != 0
				flagH := (c.F & 0x20) != 0
				flagN := (c.F & 0x40) != 0

				if !flagN {
					if flagH || (val&0x0F) > 9 {
						correction |= 0x06
					}
					if flagC || val > 0x99 {
						correction |= 0x60
						flagC = true
					}
				} else {
					if flagH {
						correction |= 0x06
					}
					if flagC {
						correction |= 0x60
					}
				}

				if flagN {
					val -= correction
				} else {
					val += correction
				}

				val &= 0xFF
				c.A = uint8(val)
				c.setFlags(val == 0, flagN, false, flagC)
			},
			Cycles: 4,
		}
	}
}
