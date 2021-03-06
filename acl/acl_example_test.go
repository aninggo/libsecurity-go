package acl_test

import (
	"fmt"

	"github.com/ibm-security-innovation/libsecurity-go/acl"
	defs "github.com/ibm-security-innovation/libsecurity-go/defs"
	en "github.com/ibm-security-innovation/libsecurity-go/entity"
)

const (
	userName1         = "User1"
	userName2         = "User2"
	resourceName      = "Camera"
	userInGroupName1  = "gUser1"
	userInGroupName2  = userName2
	groupName         = "support"
	canUsePermission  = "Can use"
	allPermission     = "All can use it"
	usersPermission   = "for users only"
	supportPermission = "Can take"
	unsetPermission   = "This permission is not allowed"
)

var (
	usersName      = []string{userName1, userName2}
	groupUsersName = []string{userInGroupName1, userInGroupName2}
)

func initEntityManager() *en.EntityManager {
	entityManager := en.New()
	for _, name := range usersName {
		entityManager.AddUser(name)
	}
	entityManager.AddGroup(groupName)
	for _, name := range groupUsersName {
		entityManager.AddUser(name)
		entityManager.AddUserToGroup(groupName, name)
	}
	entityManager.AddResource(resourceName)
	a := acl.NewACL()
	entityManager.AddPropertyToEntity(resourceName, defs.AclPropertyName, a)
	return entityManager
}

// Shows how to add/check/remove permissions for a n entity (resource) of a user or a group entity
func Example_acl() {
	entityManager := initEntityManager()
	fmt.Println("ExampleShowACLAddCheckRemovePermissions")
	fmt.Printf("User %q permission %q is %v\n", userName1, canUsePermission,
		acl.CheckUserPermission(entityManager, userName1, resourceName, en.Permission(canUsePermission)))
	data, _ := entityManager.GetPropertyAttachedToEntity(resourceName, defs.AclPropertyName)
	a, ok := data.(*acl.Acl)
	if ok == false {
		fmt.Println("Error: Cannot get property", defs.AclPropertyName, "attached to resource", resourceName)
		return
	}
	a.AddPermissionToEntity(entityManager, userName1, en.Permission(canUsePermission))
	fmt.Printf("User %q permission %q is: %v\n", userName1, canUsePermission,
		acl.CheckUserPermission(entityManager, userName1, resourceName, en.Permission(canUsePermission)))
	a.AddPermissionToEntity(entityManager, groupName, en.Permission(supportPermission))
	a.AddPermissionToEntity(entityManager, groupName, en.Permission(canUsePermission))
	a.AddPermissionToEntity(entityManager, defs.AclAllEntryName, en.Permission(allPermission))
	a.AddPermissionToEntity(entityManager, userInGroupName1, en.Permission(usersPermission))
	permissions, _ := acl.GetUserPermissions(entityManager, userInGroupName1, resourceName)
	fmt.Printf("All the permissions for user %q on resource %q are: %q\n",
		userInGroupName1, resourceName, permissions)
	permissions, _ = acl.GetUserPermissions(entityManager, groupName, resourceName)
	fmt.Printf("All the permissions for group %q on resource %q are: %q\n", groupName, resourceName, permissions)
	a.RemovePermissionFromEntity(groupName, en.Permission(canUsePermission))
	fmt.Printf("After remove permission: %q from group %q\n", canUsePermission, groupName)
	fmt.Printf("User %q permission %q is: %v\n", userInGroupName1, canUsePermission,
		acl.CheckUserPermission(entityManager, userInGroupName1, resourceName, en.Permission(canUsePermission)))
	fmt.Printf("All the permissions are: %q\n", a.GetAllPermissions())
}
