package aws_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	replacementEngine "github.com/AlexeyBeley/go_misc/replacement_engine"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type IAMAPI struct {
	svc         *iam.IAM
	DataDirPath *string
	ProfileName *string
}

func IAMAPINew(profileName *string, DataDirPath *string) *IAMAPI {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           *profileName,
	}))
	lg.Infof("AWS profile: %s\n", *profileName)
	svc := iam.New(sess)
	ret := IAMAPI{svc: svc, DataDirPath: DataDirPath, ProfileName: profileName}
	return &ret
}

func isValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

type Policy struct {
	Name     *string
	Path     *string
	Document *string
}

func (pol *Policy) SetName(name *string) error {
	if name == nil || *name == "" {
		return errors.New("policy name cannot be empty")
	}

	if len(*name) > 128 {
		return errors.New("policy name cannot exceed 128 characters")
	}
	// You can add more specific validation rules for policy names here,
	// such as allowed characters (alphanumeric, underscore, plus, dot, hyphen).
	if !regexp.MustCompile(`^[a-zA-Z0-9_+-.]+$`).MatchString(*name) {
		return errors.New("policy name contains invalid characters (only alphanumeric, _, +, ., - are allowed)")
	}

	pol.Name = name
	return nil
}

func (pol *Policy) SetPath(path *string) error {
	if path == nil {
		pol.Path = nil // Allow nil path
		return nil
	}
	if *path == "" {
		pol.Path = aws.String("/") // Default path if empty
		return nil
	}
	if len(*path) > 512 {
		return errors.New("policy path cannot exceed 512 characters")
	}
	if !regexp.MustCompile(`^(/[\w+=,.@-]+)*(/)?$`).MatchString(*path) {
		return errors.New("policy path contains invalid characters or format (must start and end with '/', and contain alphanumeric, +, =, ,, ., @, -)")
	}
	if *path != "/" && !strings.HasSuffix(*path, "/") {
		*path += "/" // Ensure path ends with '/' if not root
	}
	pol.Path = path
	return nil
}

func (pol *Policy) SetDocument(document *string) error {
	if document == nil || *document == "" {
		return errors.New("policy document cannot be empty")
	}
	if len(*document) > 131072 { // AWS IAM policy document maximum size
		return errors.New("policy document cannot exceed 131072 bytes")
	}
	// You might want to add more sophisticated validation here to ensure
	// the document is valid JSON and conforms to AWS IAM policy syntax.
	// This would likely involve parsing the JSON and checking its structure.
	// For a basic check, you can see if it's valid JSON:
	if !isValidJSON(*document) {
		return errors.New("policy document is not valid JSON")
	}
	pol.Document = document
	return nil
}

type Role struct {
	Arn            *string
	Name           *string
	Path           *string
	AssumeDocument *string
	InlinePolicies []*Policy
}

func (role *Role) SetName(name *string) error {
	role.Name = name
	return nil
}

func (role *Role) SetPath(path *string) error {
	role.Path = path
	return nil
}

func (role *Role) SetAssumeDocument(assumeDocument *string) error {
	role.AssumeDocument = assumeDocument
	return nil
}

func (role *Role) SetInlinePolicies(inlinePolicies []*Policy) error {
	role.InlinePolicies = inlinePolicies
	return nil
}

func (role *Role) UpdateFromAPIResponse(iamRole *iam.Role) error {
	role.AssumeDocument = iamRole.AssumeRolePolicyDocument
	role.Arn = iamRole.Arn
	role.Path = iamRole.Path
	return nil
}

func (api *IAMAPI) ProvisionIamRole(role *Role) (err error) {
	roleCurrent := Role{}
	roleCurrent.SetName(role.Name)
	roleCurrent.SetPath(role.Path)
	ok, err := api.UpdateRoleInfo(&roleCurrent)
	if err != nil {
		return err
	}
	if !ok {
		Input := iam.CreateRoleInput{RoleName: role.Name, Path: role.Path, AssumeRolePolicyDocument: role.AssumeDocument}
		createRoleOutput, err := api.svc.CreateRole(&Input)
		if err != nil {
			return err
		}
		lg.Infof("Created role: %v", createRoleOutput)
	}

	for _, policy := range role.InlinePolicies {
		policyInput := iam.PutRolePolicyInput{PolicyName: policy.Name, PolicyDocument: policy.Document, RoleName: role.Name}
		createPolicyOutput, err := api.svc.PutRolePolicy(&policyInput)
		if err != nil {
			return err
		}
		lg.Infof("Added Role inline policy: %v", createPolicyOutput)
	}
	return err
}

func (api *IAMAPI) UpdateRoleInfo(role *Role) (bool, error) {
	input := iam.GetRoleInput{RoleName: role.Name}
	response, err := api.svc.GetRole(&input)
	_ = response
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return false, nil
		}
		return false, err
	}

	err = role.UpdateFromAPIResponse(response.Role)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (api *IAMAPI) ProvisionIamCloudwatchWriterRole(region, roleName, strAssumeDocument, path *string) (*iam.Role, error) {
	stsAPI := STSAPINew(api.ProfileName)
	accountID, err := stsAPI.GetAccount()
	if err != nil {
		return nil, err
	}
	replacementValues := map[string]string{"STRING_REPLACEMENT_AWS_REGION": *region,
		"STRING_REPLACEMENT_AWS_ACCOUNT_ID": *accountID}
	dstDir := filepath.Join(*api.DataDirPath, "tmp")
	err = replacementEngine.ReplaceInTemplateFiles(*api.DataDirPath, dstDir, replacementValues)
	if err != nil {
		return nil, err
	}

	dstFilePath := filepath.Join(dstDir, "cloudwatch_writer_policy.json")
	document, err := os.ReadFile(dstFilePath)
	strDocument := string(document)

	if err != nil {
		fmt.Println("Error Reading file:", err)
		return nil, err
	}

	inlinePolicies := []*Policy{}
	policyName := "InlineCloudwatchWriter"

	pol := Policy{}
	pol.SetName(&policyName)
	pol.SetDocument(&strDocument)
	inlinePolicies = append(inlinePolicies, &pol)

	role := Role{}
	role.SetName(roleName)
	role.SetAssumeDocument(strAssumeDocument)
	role.SetPath(path)
	role.SetInlinePolicies(inlinePolicies)

	err = api.ProvisionIamRole(&role)

	return nil, err
}
