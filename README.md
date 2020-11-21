# dupe-finder
Because I couldn't find a good program for my usecase, I wrote this simple Go program to find duplicate files and clean them up.

## Installation

If you have go installed, the easiest way to install is `go get`:

```
go get "github.com/JaCoB1123/dupe-finder"
```

## Usage

dupe-finder supports the following options:
```
 -delete-dupes-in string
       Delete duplicates if they are contained in <path>
 -delete-prompt
       Ask which file to keep for each dupe-set
 -force
       Actually delete files. Without this options, the files to be deleted are only printed
 -from-file string
       Load results file from <path>
 -move-files string
       Move files to <path> instead of deleting them
 -to-file string
       Save results to <path>
 -verbose
       Output additional information
```

## Examples

Find all duplicate files in `~/` and save the results to `dupes.json`
```
> dupe-finder --to-file dupes.json ~/
``̀`

Load previous results from `dupes.json` and delete all duplicates located in ~/.cache
```
> dupe-finder --from-file dupes.json --delete-dupes-in ~/.cache
``̀`

Find all duplicate files in `~/' and `/mnt/EXT`. Prompt which file to keep for each set of duplicates and move the others to /dupes/.
```
> dupe-finder --delete-prompt --move-files /dupes/ ~/ /mnt/EXT
``̀`

