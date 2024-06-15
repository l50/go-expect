// Copyright 2018 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expect

import (
	"log"
	"os"
	"os/exec"
)

func ExampleConsole_echo() {
	c, err := NewConsole(WithStdout(os.Stdout))
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	cmd := exec.Command("echo")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}

	_, err = c.Send("Hello world")
	if err != nil {
		log.Println(err)
		return
	}
	_, err = c.ExpectString("Hello world")
	if err != nil {
		log.Println(err)
		return
	}
	c.Tty().Close()
	_, err = c.ExpectEOF()
	if err != nil {
		log.Println(err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		log.Println(err)
		return
	}

	// Output: Hello world
}
