package dnssd

import (
	"fmt"
	"unicode/utf8"
)

// Concatenate a three-part domain name (as provided to the response funcs) into a properly-escaped full domain name.
func ConstructFullName(serviceName, regType, domain string) string {
	return fmt.Sprintf("%s.%s.%s.", serviceName, regType, domain)
}

// domain names are "unpacked" using escape sequences and character
// escapes. Repack them to a proper UTF-8 string
func RepackToUTF8(unpacked string) string {
	us := []rune(unpacked)
	var rs []rune

	for ii := 0; ii < len(us); ii++ {
		if us[ii] == '\\' {
			ii++
			switch us[ii] {
			case 'r':
				rs = append(rs, '\r')
			case 't':
				rs = append(rs, '\t')
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				rb := make([]byte, 2)
				rb[0] = threeDigitsToInt(us[ii:])
				ii += 2
				if us[ii+1] == '\\' {
					rb[1] = threeDigitsToInt(us[ii+2:])
					r, _ := utf8.DecodeLastRune(rb)
					ii += 4
					rs = append(rs, rune(r))
				} else {
					rs = append(rs, rune(rb[0]))
				}
			default:
				rs = append(rs, us[ii])
			}
		} else {
			rs = append(rs, us[ii])
		}

	}
	return string(rs)
}

func threeDigitsToInt(us []rune) byte {
	cc := (int(us[0]) - 48) * 100
	cc += (int(us[1]) - 48) * 10
	cc += int(us[2]) - 48
	return byte(cc)
}
