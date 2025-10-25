# Census
A small tool for recursively searching your directories for quick and easy fuzzy searching.


## Setup
The easiest way to install Census is to download the [latest release](https://github.com/keystroke3/Census/releases/latest) binary.
You can then place the binary in directory that is in your system PATH, typically `/usr/bin`, `/usr/local/bin` or `~/.local/bin`.

```bash
wget -O census https://github.com/keystroke3/Census/releases/download/<latest-version>census
chmod +x census
./census --help
```

Alternatively, you can download the source code from the archive files in the releases, extract and build from source:


```bash
mkdir census
cd census
wget https://github.com/keystroke3/Census/archive/refs/tags/<verion>.tar.gz
tar -xzvf <version>.tar.gz
cd Census-<version>
go build . # add custom flags
```

In order for the build to work, you must have Golang and all the Golang tooling installed on your system.

---

**Tip**

You might want to rename the binary to something shorter like fs or set up a shell alias so you have less typing to do

---

## Usage

For a quick usage guide, just run:
```bash
census --help
```

census capabilities:
 - List all the items in the current directory
 - List all the files in multiple given directories
 - Perform REGEX filters on the search results
 - Run as a TCP server
 - Remotely call another census instance over http

### Listing

#### Simple
The main thing that census does is list items, so it is pretty easy to do that. If you want to list all the items in the 
current working directory, call census with no arguments. The paths will be listed in full from the root `-p` path
If for example we are in the directory `/pics`:

```bash
$ census
/pics/cars/ford/blue-mustang.png
/pics/cars/ford/red-fiesta.png
/pics/cars/chevrolet/silverado-truck.png
/pics/cars/chevrolet/camaro.png
/pics/animals/cats/fluffy-kitten.png
/pics/animals/cats/sleeping-tabby.png
/pics/animals/cats/playful-calico.png
/pics/animals/dogs/german-shepherd-puppy.png
/pics/animals/dogs/golden-retriever-puppy.png
/pics/animals/dogs/rottweiler-adult.png
/pics/animals/dogs/beagle-puppy.png
/pics/animals/dogs/husky-puppy.png
/pics/animals/dogs/labrador-retriever-adult.png
/pics/animals/dogs/poodle-puppy.png
```

By default, only the files are shown. If you wish to show directories instead, use the `-d` flag::

```bash
$ census -p /pics -d
/pics/cars/ford
/pics/cars/chevrolet
/pics/animals/cats
/pics/animals/dogs
```

####  Multi-directory listing

If you want to list items in more directories, you can use the `-p` or `--path` parameter for each directory you wish to add.
In the `/pics` example, you can specify
```bash
$ census -p /animals/cats  -p /pics/cars/ford
/pics/cars/ford/blue-mustang.png
/pics/cars/ford/red-fiesta.png
/pics/animals/cats/fluffy-kitten.png
/pics/animals/cats/sleeping-tabby.png
/pics/animals/cats/playful-calico.png
```

#### Piping
You can also use the output of another program as input to Census by piping into it. The paths must be separated by newline characters.
This means you can have a file that contains a list of file paths and pass them directly into Census without the need of specifying each or
using a bash variable.

```bash
$ long | pipe | chain | census
$ cat my_file_with_paths | census
```

#### Hidden Items 

By default items starting with a period '`.`' in their name are ignored unless they are explicitly included in the requested paths with `-p` parameter.
If you want them to be indexed, you can use the `-H` or `--hidden` flag. Please note that it is more performant to explicitly provide the specific
directory you wish to be included rather than enabling all the hidden directories. Especially if you have a slow drive.

Example:

```bash
$ census -p .homework
.homework/totally.png
.homework/real.png
.homework/homework.png
.homework/nothing.png
.homework/to.png
.homework/see.png
.homework/here.png

```
In this example, the extra `/pics/.homework/.secret` will not be shown.

### Filtering

#### Grep
Filtering of the results shown is done using  `-g <regexp>` or `--grep <regexp>` to *keep* items that contain `<regexp>` and `-v` or `--vgrep` to *remove* items
that contain `<regexp>`.

For example:

```bash
$ census -g 'dogs'

/pics/animals/dogs/german-shepherd-puppy.png
/pics/animals/dogs/golden-retriever-puppy.png
/pics/animals/dogs/rottweiler-adult.png
/pics/animals/dogs/beagle-puppy.png
/pics/animals/dogs/husky-puppy.png
/pics/animals/dogs/labrador-retriever-adult.png
/pics/animals/dogs/poodle-puppy.png
```

Multiple parameters can be combined in a single command:

```bash
$ census -g 'dogs' -v 'puppy'

/pics/animals/dogs/labrador-retriever-adult.png
/pics/animals/dogs/rottweiler-adult.png
```
They can also be can be repeated for extra filtering:

```bash
$ census -g 'dogs' -v 'puppy' -g 'lab'

/pics/animals/dogs/labrador-retriever-adult.png
```

The order in which the parameters are given is the order in which the filtering happens. Just as shown in the examples,
adding a second `-g` parameter does a new 'grep' operation on the results of the previous parameters. This is what happened in the last example:
 1. Find all the paths in the current directory
 2. Isolate the paths that contain 'dogs'
 3. Remove the ones that contain 'puppy' from the dogs
 4. return the labradors in the remaining non-puppy dogs

#### Ignore

You can ignore directories with `-i` or `--ignore` just as you can include them with `-p`. This might look similar to the grep parameters, but it behaves differently.
When a directory is ignored, it will not be visited at all, so it will make the file walking much faster if the directory being ignored is big. Ignoring files doesn't
have any impact, so that has not been added as an option.

#### Depth

If you have a particularly deep directory structure, and what you are looking for is relatively shallow, then it might be a good idea to limit the depth of search.
Just like ignore, it will stop searching when it reaches a certain depth and therefor save a bunch of time.

#### Trim
You can trim a prefix string from all returned file paths using the `-t` or `--trim` flag with some text to trim. If the line does not contain that prefix, it will not be touched.

### Remote Server

Suppose you have a Network Attached Storage (NAS) drive and you want to quickly fuzz out some of its contents. The simplest solution would
be to mount the NAS drive somewhere using something like SAMBA or NFS, and then run census on the mount directory. This will work
but it will be very slow and inefficient. Also, if you for some reason don't want to or can't mount the directory in question, then this might not
work for you. 

This is where the census `--serve` or `-s` parameter comes in handy. When the `--serve` parameter is passed with an addres `addr`, a new TCP listener will be started
and listen at the specified host and port. You can provide a full address like `127.0.0.1:8080` or just specify the port `:8080` and it will be assumed to be listening on localhost.
If the port is being used, then the connection will fail and the listener will not be started.

```bash
$ census --serve ':8888' 
```

Once the server is running, you can make requests to it using `--host` parameter. Everything runs just as on local machine, but all the flags and parameters are sent out to the
remote census instance where they are executed and the results are returned.

You can run the TCP server in the background like this:

```bash
$ census --serve ':8888' &> /tmp/census.log & disown
```

For a more convenient way to run it, you can define a systemd service in `/etc/systemd/system/census.service` like so:

```systemd
[Unit]
Description='census TCP server'

[Service]
User=<your_user>
ExecStart=/path/to/census -s '<ip>:<port>'

[Install]
WantedBy=multi-user.target
```
Don't for get to restart systemd daemon and enable the newly created census service so it starts at boot:

```bash
$ sudo systemctl daemon-reload
$ sudo systemctl enable census
$ sudo systemctl start census
```


---

**WARNING**

> census will walk the full directories it is instructed to if it has read access. While census does not read the contents of the files, it can be exploited by an attacker while performing
> reconnaissance to get a lay of the land they are about to attack.
> You should only use the TCP server behind a firewall in a controlled LAN environment with the port blocked form outside access. Do not expose the listening port to the wider
> internet unless you are aware of the risks and are willing to take it or have mitigations for it.
> I have some plans for adding ssh, key-pair and password support, with the last being the first to be implemented, but those are just plans for now.

___


If you wish to send the requests directly using outside tools like curl or use it scripts, you can. Just make sure the commands have the JSON format (capitalization is important):

```go
struct {
	Depth       int // required (-1 for infinite)
	DirMode     bool // false
	Grep        string // ""
	IgnorePaths []string // []
	Paths       []string // required
	ShowHidden  bool // false
	Vgrep       string // ""
}

```

Any flags or parameters not set or desired can be left out and the default values will be used.
The exception is the `Paths` and `Depth` parameters which must be provided when using external tools.  
here is an example using curl:

```bash
curl telnet://zen:10002 <<< '{"Depth":-1,"DirMode":false,"IgnorePaths":["services"],"Paths":["/media"]}'
```
`/etc/hosts` entry for `zen` has been mapped to `10.0.0.2` where census server is running.
This will also work if you have ssh hostnames defined in `~/.ssh/config`. In my case, the same machine is has the identity `zenith` in ssh config:

```bash
curl telnet://zenith:10002 <<< '{"Depth":-1,"DirMode":false,"IgnorePaths":["services"],"Paths":["/media"]}'
```

## Support
If you are happy with census and would like to support the project, here are some things you can do:

1. Tell people about the project. This will get more eyes on it and help it grow.
2. Contribute. I am open to people's contributions, be it bug fixes, new features, translations, etc.
3. Fill my coffee cup through paypal: paypal@okello.io. This will give me more energy and incentive to keep working on this tool.


DISCLAIMER:
census is a hobby project created for personal use and made public for others to use and contribute to. The authors are not responsible for any data loss or damages that may result in 
use of census.
