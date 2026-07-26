Feature: Manage locations
  As a Troventory user
  I want to create, rename, move, and archive locations
  So that I can organize my belongings in a room -> container -> shelf hierarchy

  Background:
    Given a location named "Garage" with no parent

  Scenario: Create a top-level location
    When I create a location named "Attic" with no parent
    Then the location "Attic" exists with no parent

  Scenario: Create a location nested under an existing parent
    Given a location named "Storage Rack" with parent "Garage"
    When I create a location named "Top Shelf" with parent "Storage Rack"
    Then the location "Top Shelf" exists with parent "Storage Rack"
    And "Top Shelf" is nested three levels deep under "Garage"

  Scenario: Reject creating a location under a parent that does not exist
    When I create a location named "Overflow Bin" with a parent that does not exist
    Then the location is not created
    And the request fails because the parent location cannot be found

  Scenario: Reject creating a location under a parent that has been archived
    Given the location "Garage" has been archived
    When I create a location named "Overflow Bin" with parent "Garage"
    Then the location is not created
    And the request fails because the parent location is archived

  Scenario: Reject creating a location whose name collides with a sibling under the same parent
    Given a location named "Storage Rack" with parent "Garage"
    When I create a location named "Storage Rack" with parent "Garage"
    Then the location is not created
    And the request fails because a location with that name already exists under that parent

  Scenario: Rename a location
    Given a location named "Storage Rack" with parent "Garage"
    When I rename "Storage Rack" to "Steel Shelving Unit"
    Then the location "Steel Shelving Unit" exists with parent "Garage"

  Scenario: Reject renaming a location to a name already used by a sibling under the same parent
    Given a location named "Storage Rack" with parent "Garage"
    And a location named "Workbench" with parent "Garage"
    When I rename "Workbench" to "Storage Rack"
    Then the location "Workbench" still exists with its original name
    And the request fails because a location with that name already exists under that parent

  Scenario: Move a location to a different parent
    Given a location named "Toolbox" with no parent
    And a location named "Storage Rack" with parent "Garage"
    When I move "Toolbox" to parent "Storage Rack"
    Then the location "Toolbox" exists with parent "Storage Rack"

  Scenario: Move a location to top-level
    Given a location named "Storage Rack" with parent "Garage"
    When I move "Storage Rack" to no parent
    Then the location "Storage Rack" exists with no parent

  Scenario: Reject moving a location to become a child of its own descendant
    Given a location named "Storage Rack" with parent "Garage"
    And a location named "Top Shelf" with parent "Storage Rack"
    When I move "Garage" to parent "Top Shelf"
    Then the location "Garage" still exists with its original parent
    And the request fails because the move would make the location its own descendant

  Scenario: Reject moving a location under a parent that has been archived
    Given a location named "Toolbox" with no parent
    And the location "Garage" has been archived
    When I move "Toolbox" to parent "Garage"
    Then the location "Toolbox" still exists with its original parent
    And the request fails because the parent location is archived

  Scenario: Archive a location with no active children
    Given a location named "Toolbox" with no parent
    When I archive "Toolbox"
    Then the location "Toolbox" is archived
    And "Toolbox" no longer appears among active locations

  Scenario: Reject archiving a location that still has active children
    Given a location named "Storage Rack" with parent "Garage"
    When I archive "Garage"
    Then the location "Garage" is not archived
    And the request fails because the location still has active children

  Scenario Outline: Reject an action on a location that does not exist
    When I <action> a location that does not exist
    Then the request fails because the location cannot be found

    Examples:
      | action |
      | rename |
      | move   |
      | archive |

  Scenario Outline: Reject an action on a location that has already been archived
    Given a location named "Toolbox" with no parent
    And the location "Toolbox" has been archived
    When I <action> "Toolbox"
    Then the request fails because the location is archived

    Examples:
      | action |
      | rename |
      | move   |
      | archive |

  Scenario: Submitting the same create request twice creates the location only once
    Given a create-location request for "Storage Rack" under "Garage" with reference "req-1001"
    When the request with reference "req-1001" is submitted
    And the same request with reference "req-1001" is submitted again
    Then exactly one location named "Storage Rack" exists under "Garage"
