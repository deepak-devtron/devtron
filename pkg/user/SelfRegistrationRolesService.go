package user

import (
	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/pkg/user/repository"
	"go.uber.org/zap"
)

type SelfRegistrationRolesService interface {
	GetAll() ([]string, error)
	Check() (bool, error)
	SelfRegister(emailId string)
}

type SelfRegistrationRolesServiceImpl struct {
	logger                          *zap.SugaredLogger
	selfRegistrationRolesRepository repository.SelfRegistrationRolesRepository
	userService                     UserService
}

func NewSelfRegistrationRolesServiceImpl(logger *zap.SugaredLogger,
	selfRegistrationRolesRepository repository.SelfRegistrationRolesRepository) *SelfRegistrationRolesServiceImpl {
	return &SelfRegistrationRolesServiceImpl{
		logger:                          logger,
		selfRegistrationRolesRepository: selfRegistrationRolesRepository,
	}
}

func (impl *SelfRegistrationRolesServiceImpl) GetAll() ([]string, error) {
	roleEntries, err := impl.selfRegistrationRolesRepository.GetAll()
	if err != nil {
		impl.logger.Errorf("error fetching all role %+v", err)
		return nil, err
	}
	var roles []string
	for _, role := range roleEntries {
		if role.Role != "" {
			roles = append(roles, role.Role)
		}
	}
	return roles, nil
}

func (impl *SelfRegistrationRolesServiceImpl) Check() (bool, error) {
	roleEntries, err := impl.selfRegistrationRolesRepository.GetAll()
	if err != nil {
		impl.logger.Errorf("error fetching all role %+v", err)
		return false, err
	}
	if roleEntries != nil {
		for _, role := range roleEntries {
			if role.Role != "" {
				return true, err
			}
		}
		return false, err
	}
	return false, nil
}

func (impl *SelfRegistrationRolesServiceImpl) SelfRegister(emailId string) {
	roles, err := impl.GetAll()
	if err != nil || len(roles) == 0 {
		return
	}
	userInfo := &bean.UserInfo{
		EmailId: emailId,
		Roles:   roles,
	}
	_, err = impl.userService.CreateUser(userInfo)
	if err != nil {
		impl.logger.Errorw("error while register user", "error", err)
	}
}
