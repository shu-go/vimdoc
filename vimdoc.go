package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"

	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/xnotif/util/charconv"
	"github.com/eidolon/wordwrap"
	"github.com/mattn/go-zglob"
)

type globalCmd struct {
	PkgName string `cli:"p, pkg, package"`
}

type docComment struct {
	ShortDesc string
	LongDesc  string
	SortKey   string
}

func (c docComment) String() string {
	var s string
	if c.ShortDesc != "" {
		s = "  " + c.ShortDesc + "\n"
	}
	if c.LongDesc != "" {
		s += "\n"
		s += "  " + strings.ReplaceAll(c.LongDesc, "\n", "\n  ")
	}
	return s
}

type vimFunc struct {
	Name   string
	Params []string
	docComment
}

func (f vimFunc) Signature() string {
	s := f.Name + "("
	for i, p := range f.Params {
		if i > 0 {
			s += ", "
		}
		s += "{" + p + "}"
	}
	s += ")"
	return s
}

func (f vimFunc) String() string {
	s := f.Signature()
	if f.ShortDesc != "" {
		s += "\n"
		s += f.docComment.String()
	}
	return s
}

type vimVar struct {
	Name     string
	DefValue string
	docComment
}

func (v vimVar) String() string {
	s := "let " + v.Name
	if v.DefValue != "" {
		s += " = " + v.DefValue
	}
	if v.ShortDesc != "" {
		s += "\n"
		s += v.docComment.String()
	}
	return s
}

func (g globalCmd) Run(args []string) error {
	var commentRE = regexp.MustCompile(`^\s*"""\s*(?P<comment>.*)`)

	var funcRE = regexp.MustCompile(`^\s*fu[a-z]*!?\s+(?P<name>[A-Za-z0-9#]+)\((?P<params>.*)\)`)
	var paramsepRE = regexp.MustCompile(`\s*,\s*`)

	var varRE = regexp.MustCompile(`^\s*let\s+(?P<name>g:[A-Za-z0-9]+)\s*=\s*(?P<defval>\S+)`)

	const docWidth = 78
	const specIndent = 16
	const listIndent = 32
	var specWrapeer = wordwrap.Wrapper(docWidth-specIndent, false)
	var listWrapper = wordwrap.Wrapper(docWidth-listIndent, false)

	var comment = docComment{}

	var funcs []vimFunc
	var vars []vimVar

	var files []string
	for _, a := range args {
		ff, err := zglob.Glob(a)
		if err == nil {
			files = append(files, ff...)
		}
	}

	for _, f := range files {
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

			subs := commentRE.FindStringSubmatch(line)
			if len(subs) > 1 {
				if strings.HasPrefix(strings.TrimSpace(subs[1]), "sort:") {
					comment.SortKey = strings.TrimSpace(subs[1][6:])
				} else if comment.ShortDesc == "" {
					comment.ShortDesc = subs[1]
				} else {
					if strings.TrimSpace(subs[1]) == "" {
						comment.LongDesc += "\n\n"
					} else {
						comment.LongDesc += subs[1]
					}
				}

				continue
			}

			subs = funcRE.FindStringSubmatch(line)
			if len(subs) > 0 {
				f := vimFunc{}
				if len(subs) > 1 {
					f.Name = subs[1]
				}
				if len(subs) > 2 {
					pp := paramsepRE.Split(subs[2], -1)
					if len(pp) == 1 && pp[0] == "" {
					} else {
						f.Params = pp
					}
				}

				f.docComment = comment
				funcs = append(funcs, f)

				comment = docComment{}

				continue
			}

			subs = varRE.FindStringSubmatch(line)
			if len(subs) > 0 {
				v := vimVar{}
				if len(subs) > 1 {
					v.Name = subs[1]
				}
				if len(subs) > 2 {
					v.DefValue = subs[2]
				}

				v.docComment = comment
				vars = append(vars, v)

				comment = docComment{}

				continue
			}

			if strings.TrimSpace(line) != "" {
				comment = docComment{}
			}
			//subs = commentRE.FindStringSubmatch(line)
			//if len(subs)
		}
	}

	// sort
	sort.Slice(vars, func(i, j int) bool {
		if vars[i].SortKey < vars[j].SortKey {
			return true
		} else {
			return vars[i].Name < vars[j].Name
		}
	})
	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].SortKey < funcs[j].SortKey {
			return true
		} else {
			return funcs[i].Name < funcs[j].Name
		}
	})

	if len(vars) > 0 {
		title := "VARIABLES " +
			strings.Repeat(" ", docWidth-len("VARIABLES ")-(len(g.PkgName)+len("*-variables*"))) +
			"*" + g.PkgName + "-variables*"

		fmt.Println(strings.Repeat("=", docWidth))
		fmt.Println(title)
		fmt.Println("")

		for _, v := range vars {
			if v.ShortDesc == "" {
				continue
			}

			// tag
			tag := strings.Repeat(" ", docWidth-(len(v.Name)+2)) +
				"*" + v.Name + "*"
			fmt.Println(tag)

			// name
			fmt.Println(v.Name)

			desc := specWrapeer(v.ShortDesc + "\n" + v.LongDesc)
			fmt.Println(wordwrap.Indent(desc, strings.Repeat(" ", specIndent), false))
			fmt.Println("")
		}

	}

	if len(funcs) > 0 {
		fmt.Println("")

		title := "FUNCTIONS " +
			strings.Repeat(" ", docWidth-len("FUNCTIONS ")-(len(g.PkgName)+len("*-functions*"))) +
			"*" + g.PkgName + "-functions*"

		fmt.Println(strings.Repeat("=", docWidth))
		fmt.Println(title)
		fmt.Println("")

		fmt.Println("USAGE" + strings.Repeat(" ", listIndent-len("USAGE")) + "DESCRIPTION")
		fmt.Println("")

		for _, f := range funcs {
			if f.ShortDesc == "" {
				continue
			}

			sig := f.Signature()

			sp := listIndent - len(sig)
			if sp < 0 {
				fmt.Println(sig)
				sp = listIndent
				fmt.Println(wordwrap.Indent(f.ShortDesc, strings.Repeat(" ", sp), false))
			} else {
				desc := listWrapper(f.ShortDesc)
				fmt.Println(wordwrap.Indent(desc, sig+strings.Repeat(" ", sp), false))
			}

		}
	}

	if len(funcs) > 0 {
		fmt.Println("")

		for _, f := range funcs {
			if f.LongDesc == "" {
				continue
			}

			// tag
			tag := strings.Repeat(" ", docWidth-(len(f.Name)+2+2)) +
				"*" + f.Name + "()*"
			fmt.Println(tag)

			// signature
			sig := f.Signature()
			fmt.Println(sig)

			desc := specWrapeer(f.LongDesc)
			fmt.Println(wordwrap.Indent(desc, strings.Repeat(" ", 16), false))
			fmt.Println("")
		}
	}

	fmt.Println("")

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
