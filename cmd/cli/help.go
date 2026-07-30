package main

import (
	"fmt"
	"os"
)

func printHelp(out *os.File) {
	fmt.Fprint(out, `Commands:

  location add <name> [--parent NAME]
  location rename <old-name> <new-name>
  location move <name> [--parent NAME]        (omit --parent to move to top level)
  location archive <name>
  location list                               (shows the whole hierarchy)

  item add <description> --category CAT [--date YYYY-MM-DD] [--price DOLLARS]
           [--currency USD] [--vendor NAME] [--location NAME] [--photo FILE]
  item update <description> [--description NEW] [--category CAT] [--date YYYY-MM-DD]
              [--price DOLLARS] [--currency USD] [--vendor NAME] [--location NAME]
              [--photo FILE]                  (unset flags keep the item's current value)
  item archive <description>
  item show <description>
  item scan <barcode>                         (seeds a draft item awaiting enrichment)
  item enrich <barcode> [--target DESCRIPTION] (fills gaps from a barcode/UPC lookup;
                                                omit --target to enrich the scanned draft)

  value price <description> <amount> [--currency USD] [--date YYYY-MM-DD]
  value appraise <description> <amount> [--currency USD] [--date YYYY-MM-DD]
  value depreciate <description> <rate-percent>
  value current <description> [--date YYYY-MM-DD]  (defaults to today)

  search [--desc SUBSTRING] [--category CAT] [--location NAME]
         [--min DOLLARS] [--max DOLLARS] [--currency USD]  (no flags lists everything)

  export <CSV|PDF>                            (writes the report under ./exports)

  help                                        (this message)
  exit / quit
`)
}
