package chip8

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
)

// Chip8 is the struct used for emulation
type Chip8 struct {
	memory [4096]byte

	v [16]byte

	i  uint16
	pc uint16

	delayTimer byte
	soundTimer byte

	stack [16]uint16
	sp    uint16

	opcode uint16

	drawFlag bool

	gfx [64 * 32]byte
	key [16]byte
}

var fontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xe0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0x80, // C
	0xF0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

const maxRomSize = 0xFFF - 0x200

// ErrRomTooLarge is thrown if the read in ROM size is larger than the max
var ErrRomTooLarge = errors.New("rom size is too large")

// New Loads a new version of Chip8 with fonts and the rom
func New(filepath string) (*Chip8, error) {
	// rand.Seed(time.Now().UnixNano())

	c8 := Chip8{
		pc: 0x200,
	}
	c8.loadFontset()
	err := c8.loadROM(filepath)

	if err != nil {
		return nil, err
	}

	return &c8, nil
}

func (c8 *Chip8) loadFontset() {
	for i := 0; i < 80; i++ {
		c8.memory[0] = fontSet[i]
	}
}

func (c8 *Chip8) loadROM(filepath string) error {
	rom, err := ioutil.ReadFile(filepath)

	if err != nil {
		return err
	}

	if len(rom) > maxRomSize {
		return ErrRomTooLarge
	}
	for i := 0; i < len(rom); i++ {
		c8.memory[0x200+i] = rom[i]
	}
	return nil
}

// todo: understand what this is doing
func (c8 *Chip8) drawSprite(x, y, sprite uint16) {
	c8.v[0xF] = 0
	var pix uint16

	for yLine := uint16(0); yLine < sprite; yLine++ {
		pix = uint16(c8.memory[c8.i+yLine])

		for xLine := uint16(0); xLine < 8; xLine++ {
			idx := (x + xLine + ((y + yLine) * 64))
			if idx >= uint16(len(c8.gfx)) {
				continue
			}
			if (pix & (0x80 >> xLine)) != 0 {
				if c8.gfx[idx] == 1 {
					c8.v[0xF] = 1
				}
				c8.gfx[idx] ^= 1
			}
		}
	}
	c8.drawFlag = true
}

func (c8 *Chip8) emulateCycle() {
	c8.drawFlag = false
	c8.opcode = uint16(c8.memory[c8.pc])<<8 | uint16(c8.memory[c8.pc+1])

	x := (c8.opcode & 0x0F00) >> 8
	y := (c8.opcode & 0x00F0) >> 4
	nn := byte(c8.opcode & 0x00FF) // load last 8 bits
	nnn := c8.opcode & 0x0FFF      // load last 12 bites

	switch c8.opcode & 0xF000 {
	case 0xA000:
		c8.i = c8.opcode & 0x0FFF // sets I to NNN. ANNN -> NNN
		c8.pc += 2
		break

	case 0xB000:
		c8.pc = nnn + uint16(c8.v[0])
		c8.pc += 2
		break

	case 0xC000:
		c8.v[x] = byte(rand.Intn(255)) & nn
		c8.pc += 2
		break

	case 0xD000:

		x = uint16(c8.v[x])
		y = uint16(c8.v[y])
		c8.drawSprite(x, y, c8.opcode&0x000F)
		c8.pc += 2
		break

	case 0xE000:
		switch c8.opcode & 0x00FF {
		case 0x009E:
			if c8.key[c8.v[x]] == 1 {
				c8.pc += 2
				c8.key[c8.v[x]] = 0
			}
			c8.pc += 2
			break
		case 0x00A1:
			if c8.key[c8.v[x]] == 0 {
				c8.pc += 2
			}
			c8.pc += 2
			break
		}

	case 0xF000:
		switch c8.opcode & 0x00FF {
		case 0x0007:
			c8.v[x] = c8.delayTimer
			c8.pc += 2
			break
		case 0x000A:
			for i, k := range c8.key {
				if k != 0 {
					c8.v[x] = byte(i)
					c8.pc += 2
					break
				}
			}
			c8.key[c8.v[x]] = 0
			break
		case 0x0015:
			c8.delayTimer = c8.v[x]
			c8.pc += 2
			break
		case 0x0018:
			c8.soundTimer = c8.v[x]
			c8.pc += 2
			break
		case 0x001E:
			c8.i += uint16(c8.v[x])
			c8.pc += 2
			break
		case 0x0029:
			c8.i = uint16(c8.v[x]) * 5
			c8.pc += 2
			break
		case 0x0033:
			c8.memory[c8.i] = c8.v[x] / 100
			c8.memory[c8.i+1] = (c8.v[x] / 10) % 10
			c8.memory[c8.i+2] = (c8.v[x] / 100) % 10
			c8.pc += 2
			break
		case 0x0055:
			for idx := uint16(0); idx <= x; idx++ {
				c8.memory[c8.i+idx] = c8.v[idx]
			}
			c8.pc += 2
			break
		case 0x0065:
			for idx := uint16(0); idx <= x; idx++ {
				c8.v[idx] = c8.memory[c8.i+idx]
			}
			c8.pc += 2
			break
		}

	case 0x0000:
		switch c8.opcode & 0x00FF {
		case 0x00E0: //clears the screen
			for i := 0; i < 2048; i++ {
				c8.gfx[i] = 0x0
			}
			c8.drawFlag = true
			c8.pc += 2
			break
		case 0x00EE:
			c8.pc = c8.stack[c8.sp] + 2
			c8.sp--
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c8.opcode)
			break
		}
	case 0x1000: // Jumps to address NNN
		c8.pc = nnn
		break
	case 0x2000: // Calls subroutine at NNN
		c8.sp++
		c8.stack[c8.sp] = c8.pc
		c8.pc = nnn
		break
	case 0x3000:
		if c8.v[x] == nn {
			c8.pc += 2
		}
		c8.pc += 2
		break
	case 0x4000:
		if c8.v[x] != nn {
			c8.pc += 2
		}
		c8.pc += 2
		break
	case 0x5000:
		if c8.v[x] == c8.v[y] {
			c8.pc += 2
		}
		c8.pc += 2
		break
	case 0x6000:
		c8.v[x] = nn
		c8.pc += 2
		break
	case 0x7000:
		c8.v[x] += nn
		c8.pc += 2
		break
	case 0x8000:
		switch c8.opcode & 0x000F {
		case 0x0000:
			c8.v[y] = c8.v[x]
			c8.pc += 2
			break
		case 0x0001:
			c8.v[x] = c8.v[x] | c8.v[y]
			c8.pc += 2
			break
		case 0x0002:
			c8.v[x] = c8.v[x] & c8.v[y]
			c8.pc += 2
			break
		case 0x0003:
			c8.v[x] = c8.v[x] ^ c8.v[y]
			c8.pc += 2
			break
		case 0x0004:
			if c8.v[y] > (0xFF - c8.v[x]) {
				c8.v[0xF] = 1
			} else {
				c8.v[0xF] = 0
			}
			c8.v[x] += c8.v[y]
			c8.pc += 2
			break
		case 0x0005:
			if c8.v[y] > c8.v[x] {
				c8.v[0xF] = 0
			} else {
				c8.v[0xF] = 1
			}
			c8.v[x] -= c8.v[y]
			c8.pc += 2
			break
		case 0x0006:
			vx := c8.v[x]
			c8.v[0xF] = vx & 0x1
			c8.v[x] = vx >> 1
			c8.pc += 2
			break
		case 0x0007:
			if c8.v[x] > c8.v[y] {
				c8.v[0xF] = 0
			} else {
				c8.v[0xF] = 1
			}
			c8.v[x] = c8.v[y] - c8.v[x]
			c8.pc += 2
			break
		case 0x000E:
			c8.v[0xF] = (c8.v[x] & 0x80) >> 7
			c8.v[x] = (c8.v[x] << 1) & 0xFF
			c8.pc += 2
			break
		}
	case 0x9000:
		switch c8.opcode & 0x000F {
		case 0x0000:
			if c8.v[x] != c8.v[y] {
				c8.pc += 2
			}
			c8.pc += 2
			break
		}
	default:
		fmt.Printf("Unknown opcode: 0x%X\n", c8.opcode)
	}

	// if c8.delayTimer > 0 {
	// 	c8.delayTimer--
	// }

	// if c8.soundTimer > 0 {
	// 	c8.soundTimer--
	// }
}
