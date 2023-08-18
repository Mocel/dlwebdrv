dlwebdrv
========

This command is a simple downloader that downloads the latest WebDriver for major browsers from the Internet.

## Usage

```
Usage of dlwebdrv: dlwebdrv [options] <browser>
  -channel string
        channel (default "stable")
  -debug
        debug mode
  -drivername string
        driver name
  -platform string
        platform
  -savepath string
        save path
  -v    show version
```

The `browser` argument specifies the browser for which you want to download the WebDriver.
You can specify one of the following strings:

* `chrome`
* `firefox`
* `edge`

The `-channel` option specifies the channel of the WebDriver to be downloaded. The default value is `stable`.  
This option can be specified with the following strings:

* `stable`
* `beta`
* `dev`
* `canary`

The `-platform` option specifies the operating platform for the WebDriver to be downloaded. The default value is the same as the command execution environment.  
This option can be specified with a value that combines the Go environment variables GOOS and GOARCH with a hyphen. For example, to specify the Windows x64 version, you would use `windows-amd64`.

For the values that can be specified for GOOS and GOARCH, please refer to the [Go official documentation](https://go.dev/doc/install/source#environment).

The `-drivername` option is used when you want to change the saved filename of the downloaded driver file. For example, if you specify `-drivername MicrosoftWebDriver.exe`, the downloaded file will be saved with this filename.

The `-savepath` option allows you to specify the directory where the downloaded driver file will be saved. If omitted, it will be saved in the current directory at the time of execution.


## Example

When you execute this command, it downloads the stable version of the Chrome WebDriver for Intel Mac and saves it under the `/Users/Bob/Downloads` directory.

```
$ dlwebdrv -platform darwin-amd64 -channel stable -savepath /Users/Bob/Downloads chrome
```


## License

Copyright 2024 Mocel.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
