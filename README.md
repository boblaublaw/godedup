# godedup
Golang implementation of deduplication analyzer

### File and Directory Deduplication Tool ###

This project has two goals:
 * Improve disk utilization by reducing redundant copies of files on disk.  (If this is your only goal, consider a dedup filesystem like ZFS: https://blogs.oracle.com/bonwick/entry/zfs_dedup)
 * Make it easier to confidently delete entire directories and directory trees by consolidating deduplication analysis according to user preferences.

At this time of this writing, there are several tools similar to this one on github:
 * https://github.com/hgrecco/dedup
 * https://github.com/alessandro-gentilini/keep-the-best
 * https://github.com/jpillora/dedup
 * and no doubt more...

This project is similar to several of these projects in a few ways:
 * File comparisons are made by hashing their contents.
 * Caching hash results is supported.  Modification time is examined to update cache results when needed.
 * **THIS IS BETA SOFTWARE AND YOU ASSUME ALL RESPONSIBILITY FOR MISTAKES AND/OR LOST DATA**
 * Having gotten that out of the way, this script doesn't actually delete anything.  Instead a shell script is produced, intended for review before execution.

However, unlike these other projects, this project has a few additional features I found to be absent.  See the "Features" section below for details.

## Utilization

```
godedup Usage:

        godedup <first supplied path> [additonal supplied paths ...]

        godedup will generate a human readable shell script enumerating redundant files and
        directories in the supplied paths.

Example:
        Step 1:
                godedup somedir1 somedir2 foo/bar/somedir3 > cleanup_script.sh
        Step 2:
                vi cleanup_script.sh
        Step 3:
                sh cleanup_script.sh
```

## Features

### Comparing And Removing Redundant Directories

Comparing files for redundancy is comparatively trivial.  One could achieve deduplication by ignoring directories and removing empty directories afterwards.  However, this produces a larger number of output commands.  One recursive delete accomplishes the same goal with more readability.

Thus, in cases where ```some_dir``` would be empty after deduplication, this:
```
rm -rf some_dir
```
is preferred to this:
```
rm some_dir/file1
rm some_dir/file2
...
rm some_dir/fileN
rmdir some_dir
```
Likewise, if a tree of nested directories are all empty of files after deduplicaton, the whole tree would be removed. (This can be disabled with the `-e` option.)

### Winner Selection Strategy

In cases where files or directories are deemed redundant to one another, I choose the file or directory with the shallowest directory position to be the "keeper" (or selection "winner").  Other entries which are deeper in the directory structures are slated for removal.  In cases where the depth is equal, the shorter pathname is preferred.

For example, where all the following files contain the same data:
```
somedir/file1
somedir/file10
somedir/somedir2/file2
somedir3/somedir4/somedir5/file3
```
The first file would be selected to keep and the latter three would be marked for deletion.

This strategy is effective in simplifying structures which have been copied into their own subdirectories.

### Maximizing Trust and Minimizing Error

As mentioned in the directory comparison discussion, it is my goal to simplify the generated output script to maximize the ease of review and minimize the chance of error.  To this end I try to provide shell script comments before each delete command which offer an explanation as to why it is safe to delete the candidate file or directory.

