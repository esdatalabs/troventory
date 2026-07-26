Feature: Manage the item catalog
  As a Troventory user
  I want to create, update, and archive items in my catalog
  So that I can track my belongings, their purchase details, category, and location

  Background:
    Given a location named "Garage" exists

  Scenario: Create a new item in the catalog
    When I create an item described as "Dyson V11 Vacuum" with category "Appliances", purchase date "2024-03-01", purchase price "450.00", vendor "Best Buy", assigned to location "Garage", and photos "vacuum-front.jpg, vacuum-box.jpg"
    Then the item "Dyson V11 Vacuum" exists in the catalog with category "Appliances"
    And the item "Dyson V11 Vacuum" has purchase details of date "2024-03-01", price "450.00", and vendor "Best Buy"
    And the item "Dyson V11 Vacuum" is assigned to location "Garage"
    And the item "Dyson V11 Vacuum" has photos "vacuum-front.jpg, vacuum-box.jpg"

  Scenario: Create a new item without a location assignment
    When I create an item described as "Spare Tire" with category "Automotive", purchase date "2024-01-15", purchase price "120.00", vendor "Discount Tire", and no location assigned
    Then the item "Spare Tire" exists in the catalog with no location assigned

  Scenario: Reject creating an item without a description
    When I create an item with no description, category "Appliances", purchase date "2024-03-01", purchase price "450.00", and vendor "Best Buy"
    Then the item is not created
    And the request fails because the item's description is required

  Scenario: Reject creating an item without a category
    When I create an item described as "Dyson V11 Vacuum" with no category, purchase date "2024-03-01", purchase price "450.00", and vendor "Best Buy"
    Then the item is not created
    And the request fails because the item's category is required

  Scenario: Reject creating an item assigned to a location that does not exist
    When I create an item described as "Dyson V11 Vacuum" with category "Appliances" assigned to a location that does not exist
    Then the item is not created
    And the request fails because the assigned location cannot be found

  Scenario: Reject creating an item assigned to a location that has been archived
    Given the location "Garage" has been archived
    When I create an item described as "Dyson V11 Vacuum" with category "Appliances" assigned to location "Garage"
    Then the item is not created
    And the request fails because the assigned location is archived

  Scenario: Update an existing item's details
    Given an item described as "Dyson V11 Vacuum" with category "Appliances", purchase date "2024-03-01", purchase price "450.00", and vendor "Best Buy" exists in the catalog
    And a location named "Basement" exists
    When I update "Dyson V11 Vacuum" with description "Dyson V11 Absolute Vacuum", category "Home Appliances", purchase date "2024-03-02", purchase price "475.00", vendor "Dyson Direct", assigned to location "Basement", and photos "vacuum-updated.jpg"
    Then the item "Dyson V11 Absolute Vacuum" exists in the catalog with category "Home Appliances"
    And the item "Dyson V11 Absolute Vacuum" has purchase details of date "2024-03-02", price "475.00", and vendor "Dyson Direct"
    And the item "Dyson V11 Absolute Vacuum" is assigned to location "Basement"
    And the item "Dyson V11 Absolute Vacuum" has photos "vacuum-updated.jpg"

  Scenario: Reject updating an item that does not exist
    When I update an item that does not exist
    Then the request fails because the item cannot be found

  Scenario: Reject updating an item that has been archived
    Given an item described as "Old Microwave" with category "Appliances" exists in the catalog
    And the item "Old Microwave" has been archived
    When I update "Old Microwave" with description "Old Microwave - Refurbished"
    Then the request fails because the item is archived

  Scenario: Reject updating an item's location assignment to a location that does not exist
    Given an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog
    When I update "Dyson V11 Vacuum" to be assigned to a location that does not exist
    Then the request fails because the assigned location cannot be found

  Scenario: Archive an item
    Given an item described as "Broken Toaster" with category "Appliances" exists in the catalog
    When I archive "Broken Toaster"
    Then the item "Broken Toaster" is archived
    And "Broken Toaster" no longer appears among active items in the catalog

  Scenario: Reject archiving an item that does not exist
    When I archive an item that does not exist
    Then the request fails because the item cannot be found

  Scenario: Reject archiving an item that has already been archived
    Given an item described as "Broken Toaster" with category "Appliances" exists in the catalog
    And the item "Broken Toaster" has been archived
    When I archive "Broken Toaster"
    Then the item "Broken Toaster" is not archived again
    And the request fails because the item is already archived

  Scenario: Submitting the same create request twice creates the item only once
    Given a create-item request for "Dyson V11 Vacuum" with category "Appliances" and reference "req-2001"
    When the request with reference "req-2001" is submitted
    And the same request with reference "req-2001" is submitted again
    Then exactly one item described as "Dyson V11 Vacuum" exists in the catalog
