package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
)

func checkUser(ctx context.Context, client *firestore.Client, userId string) error {
	if strings.TrimSpace(userId) == "" {
		return fmt.Errorf("no user id")
	}
	_, err := client.Collection("users").Doc(userId).Get(ctx)
	if err != nil {
		return fmt.Errorf("no such user: %v", userId)
	}
	return nil
}
func addUser(ctx context.Context, client *firestore.Client, userId string) error {
	if checkUser(ctx, client, userId) == nil {
		return nil
	}
	_, err := client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{})
	if err != nil {
		logError.Printf("Failed to create user: %v Err %v", userId, err)
		return fmt.Errorf("failed to create user: %v", userId)
	}
	return nil
}
func deleteUser(ctx context.Context, client *firestore.Client, userId string) error {
	err := checkUser(ctx, client, userId)
	if err != nil {
		return err
	}

	userDocRef := client.Collection("users").Doc(userId)

	// Delete all documents in the 'companyDetails' subcollection
	companyDetailsIter := userDocRef.Collection("companyDetails").Documents(ctx)
	companyDetailsDocs, err := companyDetailsIter.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get companyDetails documents: %v", err)
	}
	for _, doc := range companyDetailsDocs {
		_, err := doc.Ref.Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete companyDetails document: %v", err)
		}
	}

	// Add code here to delete other subcollections as needed

	// Delete the user document
	_, err = userDocRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", userId)
	}

	return nil
}

type companyDetails struct {
	ID      string `firestore:"id,omitempty"`
	Name    string `firestore:"name,omitempty"`
	Address string `firestore:"address,omitempty"`
}

func getCompanies(ctx context.Context, client *firestore.Client, userId string) ([]companyDetails, error) {

	err := checkUser(ctx, client, userId)
	if err != nil {
		return nil, err
	}
	iter := client.Collection("users").Doc(userId).Collection("companyDetails").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	clinics := make([]companyDetails, len(docs))
	for i, doc := range docs {
		var clinic companyDetails
		err := doc.DataTo(&clinic)
		if err != nil {
			return nil, err
		}
		clinics[i] = clinic
	}

	return clinics, nil
}

// add the user if it does not already exist
// Update the list with the new company list
// Remove the elements that are not in the list
func setCompanies(ctx context.Context, client *firestore.Client, userId string, companyList []companyDetails) error {
	if len(companyList) == 0 {
		return nil
	}
	addUser(ctx, client, userId)
	currentCompanies, err := getCompanies(ctx, client, userId)
	if err != nil {
		return err
	}
	retainMap := make(map[string]string)
	for _, clinic := range companyList {
		retainMap[clinic.ID] = clinic.ID
	}
	deleteItems := []string{}
	for _, clinic := range currentCompanies {
		if _, ok := retainMap[clinic.ID]; !ok {
			deleteItems = append(deleteItems, clinic.ID)
		}
	}
	bw := client.BulkWriter(ctx)

	for _, clinicId := range deleteItems {
		docRef := client.Collection("users").Doc(userId).Collection("companyDetails").Doc(clinicId)
		_, err := bw.Delete(docRef)
		if err != nil {
			return err
		}
	}
	var docRef *firestore.DocumentRef
	for _, clinic := range companyList {
		if clinic.ID == "" {
			docRef = client.Collection("users").Doc(userId).Collection("companyDetails").NewDoc()
		} else {
			docRef = client.Collection("users").Doc(userId).Collection("companyDetails").Doc(clinic.ID)
		}
		_, err := bw.Set(docRef, map[string]interface{}{
			"name":    clinic.Name,
			"address": clinic.Address,
		}, firestore.MergeAll)
		if err != nil {
			return err
		}
	}
	bw.End()
	return nil
}
