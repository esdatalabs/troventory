Feature: Search and filter items
  As a Troventory user
  I want to search and filter my items by description, category, location, and value
  So that I can quickly find what I own without scrolling through everything

  Background:
    Given a location named "Garage" with no parent
    And a location named "Storage Rack" with parent "Garage"
    And an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage" with a current value of "900.00"
    And an item described as "Craftsman Drill Set" with category "Tools" exists in the catalog assigned to location "Storage Rack" with a current value of "150.00"
    And an item described as "Vintage Lamp" with category "Home Decor" exists in the catalog assigned to location "Garage" with a current value of "75.00"
    And an item described as "Antique Pocket Watch" with category "Jewelry" exists in the catalog assigned to location "Storage Rack" with a current value of "1200.00"

  Scenario: Search for items by a partial description match
    When I search for items whose description contains "Vacuum"
    Then the search results include "Dyson V11 Vacuum"
    And the search results do not include "Craftsman Drill Set", "Vintage Lamp", or "Antique Pocket Watch"

  Scenario: Filter items by category
    When I filter items by category "Appliances"
    Then the search results include "Dyson V11 Vacuum"
    And the search results do not include "Craftsman Drill Set", "Vintage Lamp", or "Antique Pocket Watch"

  Scenario: Filtering by a location also includes items assigned to locations nested underneath it
    When I filter items by location "Garage"
    Then the search results include "Dyson V11 Vacuum" and "Vintage Lamp"
    And the search results also include "Craftsman Drill Set" and "Antique Pocket Watch", because "Storage Rack" is nested under "Garage"

  Scenario: Filtering by a more specific nested location excludes items assigned higher up the hierarchy
    When I filter items by location "Storage Rack"
    Then the search results include "Craftsman Drill Set" and "Antique Pocket Watch"
    And the search results do not include "Dyson V11 Vacuum" or "Vintage Lamp"

  Scenario: Filter items by a current value range
    When I filter items with a current value between "100.00" and "1000.00"
    Then the search results include "Dyson V11 Vacuum" and "Craftsman Drill Set"
    And the search results do not include "Vintage Lamp" or "Antique Pocket Watch"

  Scenario: Combine a description search with category and value filters
    When I search for items whose description contains "Drill" with category "Tools" and a current value between "100.00" and "500.00"
    Then the search results include exactly "Craftsman Drill Set"

  Scenario: No items match the given search
    When I search for items whose description contains "Nonexistent Widget"
    Then the search returns no results

  Scenario: Filtering by a category with no matching items returns an empty result, not an error
    When I filter items by category "Electronics"
    Then the search returns no results

  Scenario: A blank description filter matches on the other filters alone
    When I filter items by category "Appliances" with a blank description filter
    Then the search results include "Dyson V11 Vacuum"
    And the search results do not include "Craftsman Drill Set", "Vintage Lamp", or "Antique Pocket Watch"

  Scenario: Searching with no description, category, location, or value filter returns every active item
    When I search with no description, category, location, or value filter
    Then the search results include "Dyson V11 Vacuum", "Craftsman Drill Set", "Vintage Lamp", and "Antique Pocket Watch"

  Scenario: Reject a value range filter where the minimum exceeds the maximum
    When I attempt to filter items with a current value between "500.00" and "100.00"
    Then the search is rejected because the minimum value exceeds the maximum value
    And no results are returned

  Scenario: Reject filtering by a location that does not exist
    When I attempt to filter items by a location that does not exist
    Then the search is rejected because the given location cannot be found
    And no results are returned

  Scenario: Archived items are excluded from search results
    Given the item "Vintage Lamp" has been archived
    When I search for items whose description contains "Lamp"
    Then the search returns no results

  Scenario: Search results are returned in a stable, predictable order
    When I search with no description, category, location, or value filter
    Then the search results are ordered "Antique Pocket Watch", "Craftsman Drill Set", "Dyson V11 Vacuum", "Vintage Lamp"

  Scenario: Submitting the same search request twice returns identical results and is only carried out once
    Given a search request for items whose description contains "Vacuum" with reference "search-9001"
    When the request with reference "search-9001" is submitted
    And the same request with reference "search-9001" is submitted again
    Then both submissions report the same single matching item "Dyson V11 Vacuum"
    And the search is only carried out once
