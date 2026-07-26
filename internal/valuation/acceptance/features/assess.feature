Feature: Assess item value
  As a Troventory user
  I want to record purchase prices and appraisals and have depreciation applied automatically
  So that I always know the current value of each item in my inventory

  Background:
    Given an item described as "Dyson V11 Vacuum" exists in the catalog

  Scenario: Record a purchase price as the baseline valuation
    When I record a purchase price of "1200.00" purchased on "2023-01-01" for "Dyson V11 Vacuum"
    Then "Dyson V11 Vacuum" has a recorded purchase price of "1200.00" as of "2023-01-01"

  Scenario: Reject recording a purchase price that is zero or negative
    When I attempt to record a purchase price of "-50.00" purchased on "2023-01-01" for "Dyson V11 Vacuum"
    Then the purchase price is not recorded
    And the request fails because the purchase price must be a positive amount

  Scenario: Record an appraisal that becomes the item's current value
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    When I record an appraisal of "900.00" as of "2024-06-01" for "Dyson V11 Vacuum"
    Then the current value of "Dyson V11 Vacuum" as of "2024-06-01" is "900.00"

  Scenario: A later appraisal supersedes an earlier one for current-value purposes
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And an appraisal of "1000.00" was recorded for "Dyson V11 Vacuum" as of "2023-06-01"
    When I record an appraisal of "900.00" as of "2024-06-01" for "Dyson V11 Vacuum"
    Then the current value of "Dyson V11 Vacuum" as of "2024-06-01" is "900.00"

  Scenario: Reject recording an appraisal dated earlier than the most recent recorded appraisal
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And an appraisal of "900.00" was recorded for "Dyson V11 Vacuum" as of "2024-06-01"
    When I attempt to record an appraisal of "950.00" as of "2024-01-01" for "Dyson V11 Vacuum"
    Then the appraisal is not recorded
    And the request fails because the appraisal is dated earlier than the most recent recorded appraisal

  Scenario: Reject recording an appraisal that is zero or negative
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    When I attempt to record an appraisal of "0.00" as of "2024-06-01" for "Dyson V11 Vacuum"
    Then the appraisal is not recorded
    And the request fails because the appraisal value must be a positive amount

  Scenario Outline: Reject an action on an item that does not exist
    When I <action> for an item that does not exist
    Then the request fails because the item cannot be found

    Examples:
      | action                       |
      | record a purchase price      |
      | record an appraisal          |
      | compute the current value    |

  Scenario: Compute current value from the depreciated purchase price when no appraisal has been recorded
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And "Dyson V11 Vacuum" depreciates at 10% of its purchase price per year
    When I compute the current value of "Dyson V11 Vacuum" as of "2024-01-01"
    Then the current value of "Dyson V11 Vacuum" as of "2024-01-01" is "1080.00"

  Scenario: Current value retains the full purchase price when no depreciation has been configured
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And "Dyson V11 Vacuum" has no depreciation configured
    When I compute the current value of "Dyson V11 Vacuum" as of "2024-01-01"
    Then the current value of "Dyson V11 Vacuum" as of "2024-01-01" is "1200.00"

  Scenario: Depreciation continues from the most recent appraisal rather than from the original purchase price
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And "Dyson V11 Vacuum" depreciates at 10% of its purchase price per year
    And an appraisal of "1000.00" was recorded for "Dyson V11 Vacuum" as of "2024-01-01"
    When I compute the current value of "Dyson V11 Vacuum" as of "2025-01-01"
    Then the current value of "Dyson V11 Vacuum" as of "2025-01-01" is "880.00"

  Scenario: Reject computing current value for an item with no purchase price or appraisal recorded
    When I compute the current value of "Dyson V11 Vacuum" as of "2024-06-01"
    Then the request fails because no valuation has been recorded for the item

  Scenario: Submitting the same appraisal request twice records the appraisal only once
    Given "Dyson V11 Vacuum" has a purchase price of "1200.00" purchased on "2023-01-01"
    And an appraisal request of "900.00" as of "2024-06-01" for "Dyson V11 Vacuum" with reference "appraisal-7001"
    When the request with reference "appraisal-7001" is submitted
    And the same request with reference "appraisal-7001" is submitted again
    Then "Dyson V11 Vacuum" has exactly one recorded appraisal of "900.00" as of "2024-06-01"
