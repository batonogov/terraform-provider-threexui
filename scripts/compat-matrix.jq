[.versions[] | select(.supported == true) | .version]
| map(ltrimstr("v") | split(".") | map(tonumber))
| group_by(.[0:2])
| map(max | "v" + (map(tostring) | join(".")))
