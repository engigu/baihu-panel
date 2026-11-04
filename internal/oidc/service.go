package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"golang.org/x/oauth2"
)

// BuildAuthURL 生成前往单点登录提供方的授权重定向 URL
func BuildAuthURL(ctx context.Context, cfg *Config, state, verifier string) (string, error) {
	_, oauthCfg, err := GetProviderFor(ctx, cfg)
	if err != nil {
		return "", err
	}
	authURL := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return authURL, nil
}

// ExchangeAndGetClaims 完成授权码置换并返回解析后的 Claims
func ExchangeAndGetClaims(ctx context.Context, cfg *Config, code, verifier string) (*Claims, error) {
	if cfg == nil {
		return nil, fmt.Errorf("单点登录配置不能为空")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	verifierObj, oauthCfg, err := GetProviderFor(reqCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("无法连接单点登录提供方: %w", err)
	}

	token, err := oauthCfg.Exchange(reqCtx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		token, err = oauthCfg.Exchange(reqCtx, code)
	}
	if err != nil {
		return nil, fmt.Errorf("授权码换取令牌失败: %w", err)
	}

	var claims Claims
	if cfg.Type == constant.SSOTypeOAuth2 {
		cObj, err := FetchOAuth2Userinfo(reqCtx, cfg, token.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		claims = *cObj
	} else {
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			return nil, fmt.Errorf("OIDC 提供方未返回 id_token")
		}

		idToken, err := verifierObj.Verify(reqCtx, rawIDToken)
		if err != nil {
			return nil, fmt.Errorf("id_token 校验失败: %w", err)
		}
		if err := idToken.Claims(&claims); err != nil {
			return nil, fmt.Errorf("无法解析 OIDC 身份声明: %w", err)
		}
	}

	return &claims, nil
}

// ExchangeAndResolve 完成授权码置换并解析绑定系统用户（统一处理 OIDC 与 OAuth 2.0）
func ExchangeAndResolve(ctx context.Context, cfg *Config, code, verifier string, userSvc *services.UserService) (*models.User, bool, error) {
	claims, err := ExchangeAndGetClaims(ctx, cfg, code, verifier)
	if err != nil {
		return nil, false, err
	}

	user, created, err := ResolveUser(cfg, claims, userSvc)
	if err != nil {
		return nil, false, err
	}
	if user == nil {
		return nil, false, fmt.Errorf("当前第三方身份未绑定任何系统账号，且未开启自动开通")
	}

	return user, created, nil
}
