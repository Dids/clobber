[![Build Status](https://travis-ci.org/Dids/clobber.svg?branch=master)](https://travis-ci.org/Dids/clobber)

# Clobber

Clobber is command-line application for building [Clover](https://sourceforge.net/projects/cloverefiboot/).

**NOTICE:** _Work in progress. Check the list below for details._

### Implemented vs. Missing

- [x] An easy to install CLI application, distributed via `brew`  
- [x] Ability to build Clover on any machine (as long as the requirements are met)  
- [x] Target a specific Clover version/revision  
- [x] See less/more build output (`--verbose`, `--quiet` flags etc.)  
- [ ] Support for additional drivers (AptioFixPkg, ApfsSupportPkg etc.)  
- [ ] Additional customization options (select to include/exclude drivers etc.)  

### Requirements

- [macOS](https://www.apple.com/lae/macos/) (only tested on macOS High Sierra)
- [Xcode](https://developer.apple.com/xcode/) (available on the App Store)
- [Homebrew](https://brew.sh/)

Note that when you run `clobber` for the first time, it may prompt you to install [JDK](http://www.oracle.com/technetwork/java/javase/downloads/jdk8-downloads-2133151.html), saying `javac` is missing, but you can safely ignore this prompt.  
The reason for this prompt comes from building `gettext`, so it's an unfortunate side effect that we can't do anything about.

### Installation

> brew tap Dids/brewery  
> brew install clobber  

### Usage

Build the latest version of Clover:  
> clobber  

Build a specific Clover version/revision:  
> clobber --revision 1234  

View all the available options:  
> clobber --help  

### License

See [LICENSE](LICENSE).
