package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mos-nabih/lets-ride/internal/models"
	"github.com/mos-nabih/lets-ride/internal/validator"
)

type rideCreateForm struct {
	Title               string `form:"title"`
	Description         string `form:"description"`
	Departure           string `form:"departure"`
	validator.Validator `form:"-"`
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	rides, err := app.rides.Latest()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Rides = rides

	err = app.render(w, r, http.StatusOK, "home.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) rideView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	ride, err := app.rides.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
			http.NotFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Ride = ride

	err = app.render(w, r, http.StatusOK, "view.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) rideCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	data.Form = rideCreateForm{}

	err := app.render(w, r, http.StatusOK, "create.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) rideCreatePost(w http.ResponseWriter, r *http.Request) {
	var form rideCreateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Description), "description", "This field cannot be blank")
	form.CheckField(validator.MaxBytes(form.Description, 65535), "description", "This field must not be more than 65,535 bytes long")

	const departureLayout = "2006-01-02T15:04"
	departure, err := time.ParseInLocation(departureLayout, form.Departure, time.Local)
	form.CheckField(err == nil && departure.Format(departureLayout) == form.Departure, "departure", "Enter a valid departure date and time")
	form.CheckField(departure.After(time.Now()), "departure", "Departure must be in the future")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form

		err := app.render(w, r, http.StatusUnprocessableEntity, "create.tmpl", data)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	id, err := app.rides.Insert(form.Title, form.Description, departure)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Ride posted.")

	http.Redirect(w, r, fmt.Sprintf("/ride/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}

	err := app.render(w, r, http.StatusOK, "signup.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")
	form.CheckField(validator.MaxBytes(form.Password, 72), "password", "This field must not be more than 72 bytes long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form

		err := app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", data)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrDuplicateEmail):
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form

			err = app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", data)
			if err != nil {
				app.serverError(w, r, err)
			}
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}

	err := app.render(w, r, http.StatusOK, "login.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MaxBytes(form.Password, 72), "password", "This field must not be more than 72 bytes long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form

		err := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidCredentials):
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form

			err := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
			if err != nil {
				app.serverError(w, r, err)
			}
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/ride/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
