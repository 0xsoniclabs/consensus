#!/bin/bash

#
# This script adds a license header to all files in this repository.
# The license text is read from 'license_header.txt' and added at
# the beginning of each file.
# Each line of the license file is prefixed with a comment sign
# valid for respective source code.
#
# This script recognises if the header file is already present,
# and if it is same as the one in 'license_header.txt'. If the
# header is not present, or is different, the script will
# add/regenerate the header.
#
# The script can be run in two modes:
# 1. Add license headers to all files in the repository (default mode).
# 2. Check if all files have correct license headers (run with --check flag).
#

license_file="license_header.txt"

# resolve the directory of the script, no matter where it is called from
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

# resolve the root directory of the project
root_dir=$(readlink -f "$script_dir/../..")

# list of files/directories to ignore
# paths must be in single quotes to prevent shell expansion (will be expanded later)
ignore_files=('third_party/*' '*/build/*')

## extract the flag if the script should only check the license headers
check_only=false
if [[ "$#" -eq 1 && "$1" == "--check" ]]; then
    check_only=true
fi

# Extend the license text of input string, each line is prefixed.
# It is used for extending the license text of comments to be inserted
# in a source file. 
# Parameters:
#   character to use for comments, e.g.: //, #,
extend_license_header() {
    local comment_char="$1"    

    # Read the license header from the file
    local license_header=$(cat "$script_dir/$license_file")

    # Extend each line of the license header with the specified character
    local extended_license_header=""
    while IFS= read -r line; do
        extended_license_header+="$comment_char${line:+ $line}\n"
    done <<< "$license_header"

    # return
    echo "$extended_license_header"
}

# Add license header to all files in project
# root directory and all sub-directories.
# Parameters:
#   file extension, e.g.: .go, .cpp,
#   comment prefix, e.g.: //, #,
# Returns:
#   0 if all files have correct license header,
#   1 if some files have incorrect or missing license header.
add_license_to_files() {
    local file_extension="$1"
    local prefix="$2"
    local license_header="$(extend_license_header "$prefix")"
    local result=0

    # Create an array for the find command arguments
    # This approach prevents the shell from expanding the wildcard
    # characters in the ignore_files array, which would happen if
    # the script was called from a directory containing files/directories
    # that match the wildcard characters.
    local find_args=("$root_dir" -type f -name "*$file_extension")
    for pattern in "${ignore_files[@]}"; do
        find_args+=(! -path "*$pattern")
    done

    # Get a list of all files in the project directory
    local all_files=($(find "${find_args[@]}"))

    # Iterate over all files and add the license header if needed
    for f in "${all_files[@]}"; do
        # iterate over each line of the license header
        # and validate that it is present in the file
        # on the same line number, the presumption is that
        # the license header is at the beginning of the file
        local line_number=1
        local add_header=false
        while read -r line; do
            # compare the line from the license header with the line in the file on the same line number
            # whitespaces are trimmed (from the beginning and end of the line)
            if [[ "$(sed "$line_number!d" "$f" | xargs echo -n)" != "$(echo "$line" | xargs echo -n)" ]]; then
                add_header=true
                break
            fi
            line_number=$((line_number+1))
        done <<< "$(echo -e "$license_header")"

        # if the license header matched so far, check following line in the file,
        # it should be empty or contain only whitespaces
        if [[ $add_header == false ]]; then
            if [[ -n "$(sed "$line_number!d" "$f" | xargs echo -n)" ]]; then
                add_header=true
            fi
        fi

        # header should be added
        if [[ $add_header == true ]]; then
            # if the script is in check only mode, print the file name and continue
            if [[ $check_only == true ]]; then
                echo "Obsolete or missing license header in: $f"
                result=1
                continue
            fi

            # extract first line number, that does not match the license header prefix
            # in case obsolete header is present, the script will skip it
            start_from=$(grep -vnE "^$prefix" "$f" | cut -d : -f 1 | head -n 1)
            # if start_from is greater than 1, then the file contains obsolete header and we should
            # continue from `start_from + 1`, so that we don't leave double line endings
            if [[ $start_from -gt 1 ]]; then
                start_from=$((start_from+1))
            fi

            # if start_from is 1, there might be obsolete header for c++ files
            if [[ $start_from -eq 1 ]]; then
                # check if the first line is a comment beginning
                if [[ "$(sed "1!d" "$f" | xargs echo -n)" == "/*" ]]; then
                    # extract line number with the comment ending
                    start_from=$(($(grep -n "\*\/" "$f" | cut -d : -f 1 | head -n 1)+2))
                fi
            fi

            local file_content=$(tail -n +$start_from "$f")
            # add the license header to the file
            echo -e "$license_header" > "$f"
            # append the rest of the file
            echo "$file_content" >> "$f"
        fi
    done

    return $result
}

result=0
# Build files
add_license_to_files "Makefile" "#" || result=1
add_license_to_files "BUILD" "#" || result=1
# C++ files
add_license_to_files "CMakeLists.txt" "#" || result=1
add_license_to_files "*.cmake" "#" || result=1
add_license_to_files ".h" "//" || result=1
add_license_to_files ".cc" "//" || result=1
add_license_to_files ".cpp" "//" || result=1
# Go files
add_license_to_files ".go" "//" || result=1
add_license_to_files "go.mod" "//" || result=1
add_license_to_files "go.work" "//" || result=1
# Other scripts and configurations
add_license_to_files ".yml" "#" || result=1
add_license_to_files "Jenkinsfile" "//" || result=1
exit $result