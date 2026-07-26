Feature: Enrich a draft item from a barcode lookup
  As a Troventory user
  I want to scan or enter a barcode/UPC for a draft item
  So that its description, category, and photo are filled in automatically instead of by hand

  Scenario: Enrich a draft item from a valid, known barcode
    Given a draft item exists in the catalog, created from barcode "012345678905" but not yet enriched
    And the product lookup has a match for barcode "012345678905" with description "Acme Whistling Kettle", category "Kitchenware", and photo "acme-kettle.jpg"
    When I enrich the draft item using barcode "012345678905"
    Then the draft item is populated with description "Acme Whistling Kettle", category "Kitchenware", and photo "acme-kettle.jpg"

  Scenario: Barcode is well-formed but matches no known product
    Given a draft item exists in the catalog, created from barcode "099999999999" but not yet enriched
    And the product lookup has no match for barcode "099999999999"
    When I enrich the draft item using barcode "099999999999"
    Then the draft item remains unenriched
    And I am told no matching product was found for barcode "099999999999"

  Scenario: Reject a malformed barcode before attempting any lookup
    Given a draft item exists in the catalog, created from barcode "012345678905" but not yet enriched
    When I attempt to enrich the draft item using barcode "12345"
    Then the enrichment is rejected because the barcode is not a valid barcode/UPC format
    And no product lookup is attempted

  Scenario: Product lookup source is unavailable or times out
    Given a draft item exists in the catalog, created from barcode "012345678905" but not yet enriched
    And the product lookup is unavailable
    When I enrich the draft item using barcode "012345678905"
    Then the enrichment fails because the product lookup could not be completed
    And the draft item remains unenriched
    And this is reported separately from no matching product being found

  Scenario: Reject enriching an item that does not exist
    When I enrich an item that does not exist
    Then the request fails because the item cannot be found

  Scenario: Reject enriching an item that has already been archived
    Given an item described as "Old Microwave" with category "Appliances" exists in the catalog
    And the item "Old Microwave" has been archived
    When I enrich "Old Microwave" using barcode "012345678905"
    Then the request fails because the item is archived

  Scenario: Enrichment fills gaps but never overwrites details the user already entered
    Given an item described as "Vintage Lamp" with category "Home Decor" exists in the catalog with no photos
    And the product lookup has a match for barcode "012345678905" with description "Generic Table Lamp", category "Lighting", and photo "lamp.jpg"
    When I enrich "Vintage Lamp" using barcode "012345678905"
    Then the item "Vintage Lamp" still has description "Vintage Lamp" and category "Home Decor"
    And the item "Vintage Lamp" now has photo "lamp.jpg"

  Scenario: Submitting the same enrich request twice enriches the draft item only once
    Given a draft item exists in the catalog, created from barcode "012345678905" but not yet enriched
    And the product lookup has a match for barcode "012345678905" with description "Acme Whistling Kettle", category "Kitchenware", and photo "acme-kettle.jpg"
    And an enrich request for the draft item with reference "enrich-5001"
    When the request with reference "enrich-5001" is submitted
    And the same request with reference "enrich-5001" is submitted again
    Then the draft item is populated with description "Acme Whistling Kettle", category "Kitchenware", and photo "acme-kettle.jpg"
    And the item has exactly one photo, "acme-kettle.jpg", not two
