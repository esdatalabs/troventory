Feature: Export an insurance report
  As a Troventory user
  I want to compile my inventory items, their valuations, and their locations into an
  insurance report
  So that I can give my insurer proof of ownership and value for a claim

  Background:
    Given a location named "Garage" exists

  Scenario: Export a complete insurance report as a CSV document
    Given an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage"
    And "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And an appraisal of "900.00" was recorded for "Dyson V11 Vacuum" as of "2024-06-01"
    When I export an insurance report in "CSV" format
    Then the insurance report is generated in "CSV" format
    And the report lists "Dyson V11 Vacuum" with category "Appliances", purchase price "1200.00" purchased on "2023-01-01", current value "900.00" as of "2024-06-01", and location "Garage"

  Scenario: Export a complete insurance report as a PDF document
    Given an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage"
    And "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And an appraisal of "900.00" was recorded for "Dyson V11 Vacuum" as of "2024-06-01"
    When I export an insurance report in "PDF" format
    Then the insurance report is generated in "PDF" format
    And the report lists "Dyson V11 Vacuum" with category "Appliances", purchase price "1200.00" purchased on "2023-01-01", current value "900.00" as of "2024-06-01", and location "Garage"

  Scenario: Include an item that has no recorded valuation without a current value
    Given an item described as "Garden Hose" with category "Outdoor" exists in the catalog assigned to location "Garage"
    When I export an insurance report in "CSV" format
    Then the insurance report is generated in "CSV" format
    And the report lists "Garden Hose" with no current value recorded

  Scenario: Include an item that has no assigned location without a location
    Given an item described as "Spare Tire" with category "Automotive" exists in the catalog with no location assigned
    And "Spare Tire" has a purchase price of "120.00" purchased on "2023-01-01"
    When I export an insurance report in "CSV" format
    Then the insurance report is generated in "CSV" format
    And the report lists "Spare Tire" with no location recorded

  Scenario: Reject exporting a report in an unsupported format
    Given an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage"
    When I attempt to export an insurance report in "XML" format
    Then the report is not generated
    And the request fails because "XML" is not a supported export format

  Scenario: Reject exporting a report when there are no items to include
    Given the catalog contains no items
    When I attempt to export an insurance report in "CSV" format
    Then the report is not generated
    And the request fails because there are no items to include in the report

  Scenario: Archived items are excluded from the report
    Given an item described as "Broken Toaster" with category "Appliances" exists in the catalog assigned to location "Garage"
    And the item "Broken Toaster" has been archived
    And an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage"
    When I export an insurance report in "CSV" format
    Then the report lists "Dyson V11 Vacuum"
    And the report does not list "Broken Toaster"

  Scenario: Submitting the same export request twice produces the report only once
    Given an item described as "Dyson V11 Vacuum" with category "Appliances" exists in the catalog assigned to location "Garage"
    And an export request for an insurance report in "CSV" format with reference "export-3001"
    When the request with reference "export-3001" is submitted
    And the same request with reference "export-3001" is submitted again
    Then exactly one insurance report document with reference "export-3001" has been generated
