package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type result struct {
	path string
	sum  [md5.Size]byte
	err  error
}

func sumFiles(done <-chan struct{}, root string) (<-chan result, <-chan error) {
	c := make(chan result)
	errc := make(chan error, 1)
	go func() {
		var wg sync.WaitGroup
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			wg.Add(1)
			go func() {
				data, err := ioutil.ReadFile(path)
				select {
				case c <- result{path, md5.Sum(data), err}:
				case <-done:
				}
				wg.Done()
			}()
			select {
			case <-done:
				return errors.New("Walk Canceled")
			default:
				return nil
			}
		})
		go func() {
			wg.Wait()
			close(c)
		}()
		errc <- err
	}()
	return c, errc
}

func MD5All(root string) (map[string][md5.Size]byte, error) {
	done := make(chan struct{})
	defer close(done)
	c, errc := sumFiles(done, root)
	m := make(map[string][md5.Size]byte)
	for r := range c {
		if r.err != nil {
			return nil, r.err
		}
		m[r.path] = r.sum
	}
	if err := <-errc; err != nil {
		return nil, err
	}
	return m, nil
}

func hash(dir string) string {
	m, err := MD5All(dir)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	var paths []string
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	data := ""
	for _, path := range paths {
		data += fmt.Sprintf("%x", m[path])
	}
	xx := []byte(data)
	h := fmt.Sprintf("%x", md5.Sum(xx))
	return h
}

func initialize(dir string) {
	path := filepath.Join(dir, ".bt")
	inputs := filepath.Join(path, "in")
	outputs := filepath.Join(path, "out")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
		os.Mkdir(inputs, os.ModePerm)
		os.Mkdir(outputs, os.ModePerm)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func list(dir string) {
	showModel(dir, filepath.Join(dir, ".bt", "in"), "Input")
	showModel(dir, filepath.Join(dir, ".bt", "out"), "Output")
	fmt.Println("")
}

func pList(entry string, depth int) {
	indent := strings.Repeat("|   ", depth)
	fmt.Printf("%s|-- %s\n", indent, entry)
}

func showModel(rootDir string, dirname string, modelType string) {
	files, err := ioutil.ReadDir(dirname)
	check(err)
	fmt.Println("\n" + modelType + "Models:")
	if len(files) == 0 {
		fmt.Print("No Models Defined")
	}
	for _, f := range files {
		thisfile := filepath.Join(dirname, f.Name())
		dat, err := ioutil.ReadFile(thisfile)
		check(err)
		modelPath := string(dat)
		fmt.Print(" " + f.Name() + " -- ")
		if dat[0] == ([]byte("."))[0] {
			modelPath = filepath.Join(rootDir, modelPath)
		}
		fmt.Print(modelPath)
		fmt.Print("\n")
	}
}

func addInput(dir string, name string, inputPath string) {
	fmt.Println("adding input" + " " + name + " " + inputPath)
	filename := filepath.Join(dir, ".bt", "in", name)
	d1 := []byte(inputPath)
	err := ioutil.WriteFile(filename, d1, 0644)
	check(err)
}

func addOutput(dir string, name string, outputPath string) {
	fmt.Println("adding output" + " " + name + " " + outputPath)
	filename := filepath.Join(dir, ".bt", "out", name)
	d1 := []byte(outputPath)
	err := ioutil.WriteFile(filename, d1, 0644)
	check(err)
}

func removeInput(dir string, name string) {
	fmt.Println("Input\t" + name + "\tRemoved")
	filename := filepath.Join(dir, ".bt", "in", name)
	var err = os.Remove(filename)
	check(err)
}

func removeOutput(dir string, name string) {
	fmt.Println("Output\t" + name + "\tRemoved")
	filename := filepath.Join(dir, ".bt", "out", name)
	var err = os.Remove(filename)
	check(err)
}

func hlist() {
	fmt.Println(" ")
	fmt.Println("BT CLI")
	fmt.Println(" ")
	fmt.Println("Commands:\n")
	fmt.Println("init:")
	fmt.Println("  Initializes a configuration repository to the current directory")
	fmt.Println("list:")
	fmt.Println("  Views Input and Output model")
	fmt.Println("add input [name] [directory]:")
	fmt.Println("  Adds the input model to the repository")
	fmt.Println("add output [name] [directory]:")
	fmt.Println("  Adds the output model to the repository")
	fmt.Println("remove input [name]:")
	fmt.Println("  Removes the input from the repository")
	fmt.Println("remove output [name]:")
	fmt.Println("  Removes output from the directory")
	fmt.Print("hash:")
	fmt.Println("\n  Generates the hash value for the directory")
	fmt.Print("help:")
	fmt.Println("\n  Displays help menu")
	fmt.Print("log:")
	fmt.Println("\n  Displays the log of the directory")
	fmt.Print("delete:")
	fmt.Println("\n Deletes the directory")
}

func printFiles(path string, depth int) {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("error reading %s: %s\n", path, err.Error())
		return
	}
	pList(path, depth)
	for _, entry := range entries {
		if (entry.Mode() & os.ModeSymlink) == os.ModeSymlink {
			full_path, err := os.Readlink(filepath.Join(path, entry.Name()))
			if err != nil {
				fmt.Printf("error reading link: %s\n", err.Error())
			} else {
				pList(entry.Name()+" -> "+full_path, depth+1)
			}
		} else if entry.IsDir() {
			printFiles(filepath.Join(path, entry.Name()), depth+1)
		} else {
			pList(entry.Name(), depth+1)
		}
	}
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	switch os.Args[1] {
	case "init":
		initialize(dir)
	case "list":
		list(dir)
	case "add":
		switch os.Args[2] {
		case "input":
			addInput(dir, os.Args[3], os.Args[4])
		case "output":
			addOutput(dir, os.Args[3], os.Args[4])
		}
	case "remove":
		switch os.Args[2] {
		case "input":
			removeInput(dir, os.Args[3])
		case "output":
			removeOutput(dir, os.Args[3])
		}
	case "hash":
		fmt.Println(hash(dir))
	case "log":
		printFiles(".bt/out", 0)
		printFiles(".bt/in", 0)
	case "help":
		hlist()
	case "delete":
		os.RemoveAll(".bt")
	default:
		fmt.Println("Invalid Command")
		fmt.Println("\nEnter help to know commands and its functions")
	}
}
