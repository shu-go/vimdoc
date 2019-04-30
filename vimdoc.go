package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/xnotif/util/charconv"
)

type globalCmd struct{}

type docComment struct {
	ShortDesc string
	LongDesc  string
}

type vimFunc struct {
	Name   string
	Params []string
	docComment
}

type vimVar struct {
	Name     string
	DefValue string
	docComment
}

func (f vimFunc) String() string {
	s := f.Name + "("
	for i, p := range f.Params {
		if i > 0 {
			s += ", "
		}
		s += "{" + p + "}"
	}
	s += ")"
	if f.Desc != "" {
		s += "\n"
		s += "  " + f.Desc + "\n"
	}
	if f.Explanation != "" {
		s += "\n"
		s += "  " + strings.ReplaceAll(f.Explanation, "\n", "\n  ")
	}
	return s
}

func (g globalCmd) Run(args []string) error {
	var funcRE = regexp.MustCompile(`^\s*fu[a-z]*!?\s+(?P<name>[A-Za-z0-9#]+)\((?P<params>.*)\)`)
	var commentRE = regexp.MustCompile(`^\s*"""\s*(?P<comment>.*)`)
	var paramsepRE = regexp.MustCompile(`\s*,\s*`)

	funcs := []vimFunc{}
	var currFunc vimFunc

	for _, f := range args {
		content, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		str, _, err := charconv.Convert(content)
		if err != nil {
			return err
		}

		buf := bytes.NewBufferString(str)
		scanner := bufio.NewScanner(buf)

		for scanner.Scan() {
			line := scanner.Text()
			subs := funcRE.FindStringSubmatch(line)
			if len(subs) > 0 {
				if len(subs) > 1 {
					currFunc.Name = subs[1]
				}
				if len(subs) > 2 {
					pp := paramsepRE.Split(subs[2], -1)
					if len(pp) == 1 && pp[0] == "" {
					} else {
						currFunc.Params = pp
					}
				}

				continue
			}

			subs = commentRE.FindStringSubmatch(line)
			if len(subs) > 0 {
				if len(subs) > 1 {
					fmt.Printf("  %s\n", subs[1])
					if currFunc.Desc == "" {
						currFunc.Desc = subs[1]
					} else {
						if currFunc.Explanation == "" {
							currFunc.Explanation = subs[1]
						} else {
							currFunc.Explanation += "\n" + subs[1]
						}
					}
				}

				continue
			} else if currFunc.Name != "" {
				funcs = append(funcs, currFunc)
				currFunc = vimFunc{}
			}

			//subs = commentRE.FindStringSubmatch(line)
			//if len(subs)
		}

		for _, f := range funcs {
			fmt.Printf("%s\n", f)
		}
	}

	return nil
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Name = "vimdoc"
	app.Desc = ""
	app.Version = "0.0.0"
	app.Usage = `vimdoc {FILEs} > OUTPUT
vimdoc **/*.vim > output.txt
`
	app.Copyright = "(C) 2019 Shuhei Kubota"

	app.Run(os.Args)
}
